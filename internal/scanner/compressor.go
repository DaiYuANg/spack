package scanner

import (
	"context"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"path/filepath"
)

type Compressor struct {
	cacheDir   string
	algorithms []string
}

func (c *Compressor) CanProcess(mime *mimetype.MIME) bool {
	return mime.Is("application/gzip")
}
func (c *Compressor) Process(ctx context.Context, fullPath string, relPath string) error {
	for _, algo := range c.algorithms {
		// TODO: 判断是否已存在压缩文件，是否需要压缩
		targetPath := filepath.Join(c.cacheDir, relPath+"."+algo)
		fmt.Printf("Compress %s to %s\n", fullPath, targetPath)
		// 实际压缩逻辑写这里
	}
	return nil
}

func newCompressor() *Compressor {
	return &Compressor{}
}
