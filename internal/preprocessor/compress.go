package preprocessor

import (
	"io"
	"os"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

func compressZstd(src, dst string, level int) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(in *os.File) {
		_ = in.Close()
	}(in)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	enc, err := zstd.NewWriter(out, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return err
	}
	defer func(enc *zstd.Encoder) {
		_ = enc.Close()
	}(enc)

	_, err = io.Copy(enc, in)
	return err
}

func compressGzip(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(in *os.File) {
		_ = in.Close()
	}(in)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	gw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	defer func(gw *gzip.Writer) {
		_ = gw.Close()
	}(gw)

	if err != nil {
		return err
	}

	_, err = io.Copy(gw, in)
	return err
}

func compressBrotli(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func(in *os.File) {
		_ = in.Close()
	}(in)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		_ = out.Close()
	}(out)

	bw := brotli.NewWriterLevel(out, brotli.BestCompression)
	defer func(bw *brotli.Writer) {
		_ = bw.Close()
	}(bw)

	_, err = io.Copy(bw, in)
	return err
}
