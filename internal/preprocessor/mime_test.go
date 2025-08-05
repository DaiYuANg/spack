package preprocessor_test

import (
	"github.com/gabriel-vasile/mimetype"
	"mime"
	"testing"
)

func TestMimetype(t *testing.T) {
	println("test")
	mimeT := mime.TypeByExtension(".css")
	println(mimeT)
	m := mimetype.Lookup("text/css")
	println(m)
}
