package source

import "time"

type File struct {
	Path     string
	FullPath string
	Size     int64
	IsDir    bool
	ModTime  time.Time
}

type Source interface {
	Walk(func(File) error) error
}
