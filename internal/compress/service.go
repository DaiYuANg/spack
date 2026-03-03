package compress

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/model"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"golang.org/x/sync/singleflight"
)

type Task struct {
	SourceKey string
	Encodings []string
}

type Service struct {
	cfg      *config.Compression
	registry registry.Registry
	logger   *slog.Logger

	tasks   chan Task
	stopCh  chan struct{}
	wg      sync.WaitGroup
	sf      singleflight.Group
	workers int
}

func newService(
	lc fx.Lifecycle,
	cfg *config.Compression,
	reg registry.Registry,
	logger *slog.Logger,
) *Service {
	workers := cfg.Workers
	if workers < 1 {
		workers = 1
	}

	svc := &Service{
		cfg:      cfg,
		registry: reg,
		logger:   logger,
		tasks:    make(chan Task, workers*64),
		stopCh:   make(chan struct{}),
		workers:  workers,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !cfg.Enable {
				logger.Info("Compression disabled")
				return nil
			}
			if strings.TrimSpace(cfg.CacheDir) == "" {
				return fmt.Errorf("compression cache dir is empty")
			}
			if err := os.MkdirAll(cfg.CacheDir, 0o755); err != nil {
				return err
			}
			svc.startWorkers()
			logger.Info("Compression workers started",
				slog.String("cache_dir", cfg.CacheDir),
				slog.Int("workers", svc.workers),
				slog.Int64("min_size", cfg.MinSize),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			close(svc.stopCh)
			done := make(chan struct{})
			go func() {
				svc.wg.Wait()
				close(done)
			}()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})

	return svc
}

func (s *Service) Enqueue(sourceKey string, encodings []string) {
	if !s.cfg.Enable || strings.TrimSpace(sourceKey) == "" {
		return
	}
	normalized := normalizeEncodings(encodings)
	if len(normalized) == 0 {
		return
	}

	task := Task{
		SourceKey: sourceKey,
		Encodings: normalized,
	}

	select {
	case s.tasks <- task:
	default:
		s.logger.Debug("Compression queue full, drop task", slog.String("source_key", sourceKey))
	}
}

func (s *Service) startWorkers() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go func(id int) {
			defer s.wg.Done()
			for {
				select {
				case <-s.stopCh:
					return
				case task := <-s.tasks:
					s.processTask(task)
				}
			}
		}(i)
	}
}

func (s *Service) processTask(task Task) {
	source, err := s.registry.FindByKey(task.SourceKey)
	if err != nil {
		s.logger.Debug("Compression task source not found", slog.String("source_key", task.SourceKey))
		return
	}
	if !s.shouldCompress(source) {
		return
	}

	sourceHash := s.sourceHash(source)
	if sourceHash == "" {
		hash, hashErr := hashFile(source.FullPath)
		if hashErr != nil {
			s.logger.Error("Compression source hash failed", slog.String("source_key", source.Key), slog.String("err", hashErr.Error()))
			return
		}
		sourceHash = hash
		if source.Metadata == nil {
			source.Metadata = map[string]string{}
		}
		source.Metadata["source_hash"] = sourceHash
		if source.Metadata["etag"] == "" {
			source.Metadata["etag"] = fmt.Sprintf("\"%s\"", sourceHash)
		}
		if err := s.registry.Register(source); err != nil {
			s.logger.Error("Compression source metadata sync failed", slog.String("source_key", source.Key), slog.String("err", err.Error()))
			return
		}
	}

	lo.ForEach(task.Encodings, func(enc string, _ int) {
		if err := s.ensureVariant(source, sourceHash, enc); err != nil {
			s.logger.Error("Compression generate failed",
				slog.String("source_key", source.Key),
				slog.String("encoding", enc),
				slog.String("err", err.Error()),
			)
		}
	})
}

func (s *Service) ensureVariant(source *model.ObjectInfo, sourceHash, encoding string) error {
	sfKey := source.Key + "|" + sourceHash + "|" + encoding
	_, err, _ := s.sf.Do(sfKey, func() (interface{}, error) {
		if s.variantReady(source, sourceHash, encoding) {
			return nil, nil
		}

		raw, err := os.ReadFile(source.FullPath)
		if err != nil {
			return nil, err
		}
		if int64(len(raw)) < s.cfg.MinSize {
			return nil, nil
		}

		compressed, ext, err := s.compress(raw, encoding)
		if err != nil {
			return nil, err
		}
		if len(compressed) >= len(raw) {
			return nil, nil
		}

		targetPath, variantKey := s.variantLocation(source.Key, sourceHash, ext)
		if err := writeFileAtomic(targetPath, compressed); err != nil {
			return nil, err
		}
		info, err := os.Stat(targetPath)
		if err != nil {
			return nil, err
		}

		variant := &model.ObjectInfo{
			Key:      variantKey,
			Size:     info.Size(),
			FullPath: targetPath,
			Reader: func() (io.ReadCloser, error) {
				return os.Open(targetPath)
			},
			IsDir:    false,
			Mimetype: source.Mimetype,
			Metadata: map[string]string{
				"encoding":    encoding,
				"source_key":  source.Key,
				"source_hash": sourceHash,
				"etag":        fmt.Sprintf("\"%s-%s\"", sourceHash, encoding),
				"mtime_unix":  strconv.FormatInt(info.ModTime().Unix(), 10),
				"generated":   "true",
			},
		}

		if err := s.registry.Register(variant); err != nil {
			return nil, err
		}
		if err := s.registry.RegisterChildren(source, variant); err != nil {
			return nil, err
		}
		if err := s.registry.RegisterParents(variant, source); err != nil {
			return nil, err
		}

		s.logger.Debug("Compression variant ready",
			slog.String("source_key", source.Key),
			slog.String("variant_key", variant.Key),
			slog.String("path", targetPath),
			slog.String("encoding", encoding),
		)

		return nil, nil
	})
	return err
}

func (s *Service) variantReady(source *model.ObjectInfo, sourceHash, encoding string) bool {
	children, err := s.registry.ListChildren(source.Key)
	if err != nil {
		return false
	}

	return lo.ContainsBy(children, func(child *model.ObjectInfo) bool {
		if !variantMatchesEncoding(child.Key, source.Key, encoding) {
			return false
		}
		if child.Metadata != nil {
			if childEncoding := strings.ToLower(strings.TrimSpace(child.Metadata["encoding"])); childEncoding != "" && childEncoding != encoding {
				return false
			}
			if childSourceHash := strings.TrimSpace(child.Metadata["source_hash"]); childSourceHash != "" && sourceHash != "" && childSourceHash != sourceHash {
				return false
			}
		}
		if child.FullPath == "" {
			return false
		}
		if _, statErr := os.Stat(child.FullPath); statErr != nil {
			return false
		}
		return true
	})
}

func (s *Service) sourceHash(source *model.ObjectInfo) string {
	if source == nil || source.Metadata == nil {
		return ""
	}
	return strings.TrimSpace(source.Metadata["source_hash"])
}

func (s *Service) variantLocation(sourceKey, sourceHash, ext string) (string, string) {
	cleanKey := filepath.FromSlash(strings.TrimPrefix(sourceKey, "/"))
	targetPath := filepath.Join(s.cfg.CacheDir, sourceHash, cleanKey+"."+ext)
	variantKey := sourceKey + "." + ext
	return targetPath, variantKey
}

func (s *Service) compress(raw []byte, encoding string) ([]byte, string, error) {
	switch encoding {
	case "br":
		var buf bytes.Buffer
		writer := brotli.NewWriterLevel(&buf, clampBrotliQuality(s.cfg.BrotliQuality))
		if _, err := writer.Write(raw); err != nil {
			return nil, "", err
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "br", nil
	case "gzip":
		var buf bytes.Buffer
		writer, err := gzip.NewWriterLevel(&buf, clampGzipLevel(s.cfg.GzipLevel))
		if err != nil {
			return nil, "", err
		}
		if _, err := writer.Write(raw); err != nil {
			return nil, "", err
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "gz", nil
	default:
		return nil, "", fmt.Errorf("unsupported encoding: %s", encoding)
	}
}

func (s *Service) shouldCompress(source *model.ObjectInfo) bool {
	if source == nil || source.IsDir {
		return false
	}
	if strings.TrimSpace(source.FullPath) == "" {
		return false
	}
	if source.Size < s.cfg.MinSize {
		return false
	}
	mime := strings.ToLower(source.MimeString())
	if mime == "" {
		return false
	}
	if strings.HasPrefix(mime, "text/") {
		return true
	}
	if strings.HasPrefix(mime, "image/") && mime != "image/svg+xml" {
		return false
	}
	if strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/") {
		return false
	}
	if strings.Contains(mime, "zip") || strings.Contains(mime, "gzip") {
		return false
	}
	switch mime {
	case "application/javascript",
		"application/json",
		"application/wasm",
		"application/xml",
		"image/svg+xml":
		return true
	default:
		return false
	}
}

func normalizeEncodings(encodings []string) []string {
	if len(encodings) == 0 {
		return nil
	}

	normalized := lo.Map(encodings, func(raw string, _ int) string {
		enc := strings.ToLower(strings.TrimSpace(raw))
		return enc
	})
	normalized = lo.Filter(normalized, func(enc string, _ int) bool {
		return enc == "br" || enc == "gzip"
	})
	if len(normalized) == 0 {
		return nil
	}

	return lo.Uniq(normalized)
}

func variantMatchesEncoding(childKey, sourceKey, encoding string) bool {
	switch encoding {
	case "br":
		return strings.TrimSuffix(childKey, ".br") == sourceKey && strings.HasSuffix(childKey, ".br")
	case "gzip":
		return strings.TrimSuffix(childKey, ".gz") == sourceKey && strings.HasSuffix(childKey, ".gz")
	default:
		return false
	}
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func clampGzipLevel(level int) int {
	if level < gzip.BestSpeed || level > gzip.BestCompression {
		return gzip.DefaultCompression
	}
	return level
}

func clampBrotliQuality(q int) int {
	if q < 0 {
		return 0
	}
	if q > 11 {
		return 11
	}
	return q
}
