package finder

import "github.com/daiyuang/spack/internal/constant"

type Result struct {
	MediaType constant.MimeType
	Data      []byte
}

func (r Result) MediaTypeString() string {
	return string(r.MediaType)
}
