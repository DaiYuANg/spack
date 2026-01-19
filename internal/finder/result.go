package finder

import "github.com/daiyuang/spack/internal/constant"

type Result struct {
	Key            string
	MediaType      constant.MimeType
	Data           []byte
	Encoding       string // "", "gzip", "br"
	AcceptEncoding []string
}

func (r Result) MediaTypeString() string {
	return string(r.MediaType)
}
