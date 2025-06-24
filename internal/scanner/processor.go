package scanner

import (
	"context"
	"github.com/gabriel-vasile/mimetype"
)

type FileProcessor interface {
	// Process 处理单个文件（绝对路径 + 相对路径）
	Process(ctx context.Context, fullPath string, relPath string) error

	CanProcess(mime *mimetype.MIME) bool
}
