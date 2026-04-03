package pipeline

import (
	"slices"
	"testing"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func TestNormalizeEncodingsPreservesFirstSeenOrder(t *testing.T) {
	got := normalizeEncodings(collectionx.NewList(" gzip ", "br", "gzip", "bad", "br"))
	want := []string{"gzip", "br"}
	if !slices.Equal(got.Values(), want) {
		t.Fatalf("expected encodings %#v, got %#v", want, got)
	}
}

func TestNormalizeImageFormatsPreservesFirstSeenOrder(t *testing.T) {
	got := normalizeImageFormats(collectionx.NewList(" png ", "jpeg", "jpg", "png", "bad"))
	want := []string{"png", "jpeg"}
	if !slices.Equal(got.Values(), want) {
		t.Fatalf("expected formats %#v, got %#v", want, got)
	}
}

func TestNormalizeRequestStringsSortsAndDeduplicates(t *testing.T) {
	got := normalizeRequestStrings(collectionx.NewList(" gzip ", "br", "gzip", "", " BR "))
	want := []string{"br", "gzip"}
	if !slices.Equal(got.Values(), want) {
		t.Fatalf("expected request strings %#v, got %#v", want, got)
	}
}

func TestNormalizeRequestIntsSortsAndDeduplicates(t *testing.T) {
	got := normalizeRequestInts(collectionx.NewList(1280, 640, 1280, 0, -1))
	want := []int{640, 1280}
	if !slices.Equal(got.Values(), want) {
		t.Fatalf("expected request ints %#v, got %#v", want, got)
	}
}
