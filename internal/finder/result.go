package finder

import "github.com/daiyuang/spack/internal/constant"

type Result struct {
	Key            string
	MediaType      constant.MimeType
	Data           []byte
	Encoding       string   // "", "gzip", "br"
	AcceptEncoding []string // preferred encodings parsed from request header
	ETag           string
}

func (r Result) MediaTypeString() string {
	return string(r.MediaType)
}
