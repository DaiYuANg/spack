package config

import (
	"slices"
	"testing"
)

func TestImageParsedWidthsFiltersSortsAndDeduplicates(t *testing.T) {
	cfg := Image{Widths: "1280, 640, bad, 1280, 0, -1, 1920"}

	got := cfg.ParsedWidths()
	want := []int{640, 1280, 1920}
	if !slices.Equal(got.Values(), want) {
		t.Fatalf("expected widths %#v, got %#v", want, got)
	}
}
