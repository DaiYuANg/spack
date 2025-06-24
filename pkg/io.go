package pkg

import (
	"github.com/gabriel-vasile/mimetype"
	"os"
)

func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func DetectMime(path string) (*mimetype.MIME, error) {
	mimetype.SetLimit(0)
	mimeType, err := mimetype.DetectFile(path)
	if err != nil {
		return nil, err
	}
	return mimeType, err
}
