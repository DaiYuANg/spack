package preprocessor

import (
	"bytes"
	"github.com/chai2010/webp"
	"image"
	"log"
)

type imagePreprocessor struct {
}

func (receiver *imagePreprocessor) name(data []byte) image.Image {
	m, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		log.Println(err)
	}
	return m
}
