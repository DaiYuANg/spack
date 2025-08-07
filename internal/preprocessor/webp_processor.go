package preprocessor

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type webpPreprocessor struct {
	logger      *zap.SugaredLogger
	supportMime []constant.MimeType
	registry    registry.Registry
}

func (w *webpPreprocessor) Name() string {
	return "webp"
}

func (w *webpPreprocessor) Order() int {
	return 0
}

func (w *webpPreprocessor) CanProcess(info *registry.OriginalFileInfo) bool {
	ok := lo.ContainsBy(w.supportMime, func(mt constant.MimeType) bool {
		return string(mt) == info.Mimetype
	})

	if ok {
		w.logger.Debugf("webp: matched mime=%s for path=%s", info.Mimetype, info.Path)
	}

	return ok
}
func (w *webpPreprocessor) getCacheDir() (string, error) {
	version := pkg.GetVersionFromBuildInfo()
	basePath := filepath.Join(os.TempDir(), version)
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return "", err
	}
	return basePath, nil
}

func (w *webpPreprocessor) generateTargetPath(info *registry.OriginalFileInfo) (string, error) {
	cacheDir, err := w.getCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, info.Hash+".webp"), nil
}

func (w *webpPreprocessor) cacheExists(targetPath string) bool {
	_, err := os.Stat(targetPath)
	return err == nil
}

func (w *webpPreprocessor) loadImage(path string) (image.Image, error) {
	srcFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func(srcFile *os.File) {
		_ = srcFile.Close()
	}(srcFile)

	img, _, err := image.Decode(srcFile)
	return img, err
}

func (w *webpPreprocessor) encodeWebp(img image.Image, targetPath string) error {
	outFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer func(outFile *os.File) {
		_ = outFile.Close()
	}(outFile)

	return webp.Encode(outFile, img, &webp.Options{Lossless: true, Quality: 80})
}

func (w *webpPreprocessor) Process(info *registry.OriginalFileInfo) error {
	path := info.Path
	targetPath, err := w.generateTargetPath(info)
	if err != nil {
		w.logger.Errorf("failed to generate target path for %s: %v", path, err)
		return err
	}

	if w.cacheExists(targetPath) {
		w.logger.Debugf("webp: cached file exists %s, register variant", targetPath)

		stat, err := os.Stat(targetPath)
		if err != nil {
			return err
		}

		vinfo := &registry.VariantFileInfo{
			Path:        targetPath,
			VariantType: constant.VariantWebP,
			Size:        stat.Size(),
			Ext:         ".webp",
		}
		w.registry.AddVariant(info.Path, vinfo)
		return nil
	}

	w.logger.Debugf("webp: converting %s to %s", path, targetPath)

	img, err := w.loadImage(path)
	if err != nil {
		return err
	}

	if err := w.encodeWebp(img, targetPath); err != nil {
		return err
	}

	stat, err := os.Stat(targetPath)
	if err != nil {
		return err
	}

	vinfo := &registry.VariantFileInfo{
		Path:        targetPath,
		VariantType: constant.VariantWebP,
		Size:        stat.Size(),
		Ext:         ".webp",
	}
	w.registry.AddVariant(info.Path, vinfo)

	w.logger.Debugf("webp: generated and registered %s", targetPath)
	return nil
}

func newWebpPreprocessor(logger *zap.SugaredLogger, r registry.Registry) *webpPreprocessor {
	return &webpPreprocessor{
		logger:      logger,
		supportMime: []constant.MimeType{constant.Png, constant.Jpg, constant.Jpeg},
		registry:    r,
	}
}
