package scanner

import (
	"context"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/samber/lo"
	"go.etcd.io/bbolt"
	"go.uber.org/fx"
	"path/filepath"
	"strings"
)

var support = []string{
	"image/jpeg",
	"image/png",
	"image/gif",
}

type WebPConverter struct {
	output string
}

func (c *WebPConverter) CanProcess(mime *mimetype.MIME) bool {
	var filtered = lo.Filter(support, func(item string, index int) bool {
		return mime.Is(item)
	})
	return len(filtered) != 0
}

func (c *WebPConverter) Process(ctx context.Context, fullPath string, relPath string) error {
	// 判断缓存中是否已有对应的 WebP 文件，是否需要重新生成
	// 生成 WebP 文件到缓存目录
	// 简单示例：
	targetPath := filepath.Join(c.output, strings.TrimSuffix(relPath, filepath.Ext(relPath))+".webp")
	// TODO: 实际转换逻辑
	fmt.Printf("Convert %s to %s\n", fullPath, targetPath)
	return nil
}

type NewWebpConverterDependency struct {
	fx.In
	BaseDir string `name:"baseDir"`
	DB      *bbolt.DB
}

func newWebPConverter(dep NewWebpConverterDependency) *WebPConverter {
	return &WebPConverter{
		output: filepath.Join(dep.BaseDir, "webp"),
	}
}
