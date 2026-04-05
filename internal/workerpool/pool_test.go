package workerpool_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/daiyuang/spack/internal/config"
	"github.com/daiyuang/spack/internal/workerpool"
)

func TestNewSettingsUsesNormalizedAsyncWorkers(t *testing.T) {
	settings := workerpool.NewSettingsForTest(&config.Async{Workers: 5})
	if settings.Size != 5 {
		t.Fatalf("expected settings size 5, got %d", settings.Size)
	}
}

func TestNewPoolUsesConfiguredSize(t *testing.T) {
	pool, err := workerpool.NewPoolForTest(&workerpool.Settings{Size: 3})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := pool.ReleaseTimeout(3 * time.Second); cleanupErr != nil {
			t.Fatalf("release worker pool: %v", cleanupErr)
		}
	})

	if got := pool.Cap(); got != 3 {
		t.Fatalf("expected ants pool cap 3, got %d", got)
	}
}

func TestRunListFallsBackToSerialWithoutPool(t *testing.T) {
	values := collectionx.NewList(1, 2, 3)
	visited := make([]int, 0, values.Len())

	err := workerpool.RunList(context.Background(), nil, values, func(_ context.Context, value int) error {
		visited = append(visited, value)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(visited, []int{1, 2, 3}) {
		t.Fatalf("expected serial visit order [1 2 3], got %v", visited)
	}
}

func TestRunListUsesPool(t *testing.T) {
	pool, err := workerpool.NewPoolForTest(&workerpool.Settings{Size: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := pool.ReleaseTimeout(3 * time.Second); cleanupErr != nil {
			t.Fatalf("release worker pool: %v", cleanupErr)
		}
	})

	values := collectionx.NewList(1, 2, 3, 4)
	visited := collectionx.NewConcurrentSet[int]()

	err = workerpool.RunList(context.Background(), pool, values, func(_ context.Context, value int) error {
		visited.Add(value)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	got := visited.Values()
	slices.Sort(got)
	if !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Fatalf("expected all values visited once, got %v", got)
	}
}
