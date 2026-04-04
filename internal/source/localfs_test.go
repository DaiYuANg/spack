package source_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/source"
)

func TestNewLocalFSRequiresRoot(t *testing.T) {
	_, err := source.NewLocalFSForTest(&config.Assets{}, slog.New(slog.DiscardHandler))
	if err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestNewLocalFSRequiresDirectory(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "spack-file-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
	}()

	_, err = source.NewLocalFSForTest(&config.Assets{Root: file.Name()}, slog.New(slog.DiscardHandler))
	if err == nil {
		t.Fatal("expected error for file root")
	}
}
