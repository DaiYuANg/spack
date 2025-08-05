package preprocessor

import (
	"github.com/chai2010/webp"
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/pkg"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
)

type webpPreprocessor struct {
	logger      *zap.SugaredLogger
	supportMime []constant.MimeType
}

func (w *webpPreprocessor) Name() string {
	return "webp"
}

func (w *webpPreprocessor) Order() int {
	return 0
}

func (w *webpPreprocessor) CanProcess(path string, mimetype string) bool {
	ok := lo.ContainsBy(w.supportMime, func(mt constant.MimeType) bool {
		return string(mt) == mimetype
	})

	if ok {
		w.logger.Debugf("webp: matched mime=%s for path=%s", mimetype, path)
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

func (w *webpPreprocessor) generateTargetPath(originalPath string) (string, error) {
	hash, err := pkg.FileHash(originalPath)
	if err != nil {
		return "", err
	}

	cacheDir, err := w.getCacheDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(cacheDir, hash+".webp"), nil
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

func (w *webpPreprocessor) Process(path string) error {
	targetPath, err := w.generateTargetPath(path)
	if err != nil {
		w.logger.Errorf("failed to generate target path for %s: %v", path, err)
		return err
	}

	if w.cacheExists(targetPath) {
		w.logger.Debugf("webp: cached file exists %s, skip conversion", targetPath)
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

	w.logger.Debugf("webp: generated %s", targetPath)
	return nil
}

func newWebpPreprocessor(logger *zap.SugaredLogger) *webpPreprocessor {
	return &webpPreprocessor{
		logger:      logger,
		supportMime: []constant.MimeType{constant.Png, constant.Jpg, constant.Jpeg},
	}
}
