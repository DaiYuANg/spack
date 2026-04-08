package media_test

import (
	"testing"

	"github.com/daiyuang/spack/internal/media"
)

func TestIsTextLikeMediaType(t *testing.T) {
	tests := []struct {
		mediaType string
		want      bool
	}{
		{mediaType: "text/html; charset=utf-8", want: true},
		{mediaType: "application/javascript", want: true},
		{mediaType: "application/manifest+json", want: true},
		{mediaType: "image/svg+xml", want: true},
		{mediaType: "application/vnd.api+json", want: true},
		{mediaType: "image/png", want: false},
	}

	for _, tt := range tests {
		if got := media.IsTextLikeMediaType(tt.mediaType); got != tt.want {
			t.Fatalf("IsTextLikeMediaType(%q) = %v, want %v", tt.mediaType, got, tt.want)
		}
	}
}

func TestIsCompressibleMediaType(t *testing.T) {
	tests := []struct {
		mediaType string
		want      bool
	}{
		{mediaType: "text/css", want: true},
		{mediaType: "application/javascript", want: true},
		{mediaType: "application/wasm", want: true},
		{mediaType: "image/svg+xml", want: true},
		{mediaType: "image/png", want: false},
		{mediaType: "application/gzip", want: false},
		{mediaType: "", want: false},
	}

	for _, tt := range tests {
		if got := media.IsCompressibleMediaType(tt.mediaType); got != tt.want {
			t.Fatalf("IsCompressibleMediaType(%q) = %v, want %v", tt.mediaType, got, tt.want)
		}
	}
}
