package registry

import (
	"github.com/daiyuang/spack/internal/constant"
)

type jsonOriginal struct {
	Path     string            `json:"path"`
	MIME     constant.MimeType `json:"mime"`
	Size     int64             `json:"size"`
	Hash     string            `json:"hash"`
	VarCount int               `json:"variant_count"`
}

type jsonReport struct {
	Originals []*jsonOriginal                       `json:"originals"`
	ByMIME    map[constant.MimeType][]*jsonOriginal `json:"by_mime"`
	Total     int                                   `json:"total"`
}
