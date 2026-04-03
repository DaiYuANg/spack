package source

import (
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/daiyuang/spack/internal/config"
)

func TestNewLocalFSRequiresRoot(t *testing.T) {
	_, err := newLocalFS(&config.Assets{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestNewLocalFSRequiresDirectory(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "spack-file-*")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	_, err = newLocalFS(&config.Assets{Root: file.Name()}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error for file root")
	}
}
