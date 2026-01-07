package registry

import (
	"github.com/daiyuang/spack/internal/constant"
	"github.com/daiyuang/spack/internal/storage"
)

type jsonOriginal struct {
	Path     string            `json:"path"`
	MIME     constant.MimeType `json:"mime"`
	Size     int64             `json:"size"`
	Hash     string            `json:"hash"`
	Variants []*jsonVariant    `json:"variants"`
	VarCount int               `json:"variant_count"`
}

type jsonVariant struct {
	Ext         string      `json:"ext"`
	VariantType string      `json:"variant_type"`
	Size        int64       `json:"size"`
	StorageKey  storage.Key `json:"storage_key"`
}

type jsonReport struct {
	Originals []*jsonOriginal                       `json:"originals"`
	ByMIME    map[constant.MimeType][]*jsonOriginal `json:"by_mime"`
	Total     int                                   `json:"total"`
}
