package cache

import (
	"mime"
	"os"
	"path/filepath"
	"time"
)

type CachedFile struct {
	Content     []byte
	ContentType string
	ModTime     time.Time
	Size        int64
}

func LoadFileToCache(path string) (*CachedFile, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return &CachedFile{
		Content:     data,
		ContentType: contentType,
		ModTime:     info.ModTime(),
		Size:        info.Size(),
	}, nil
}
