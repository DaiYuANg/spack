package registry

import (
	"fmt"
	"sort"
	"sync"

	"github.com/samber/lo"
	"github.com/samber/oops"
)

type memoryRegistry struct {
	// 构建期间可写的原始数据结构
	mu        sync.Mutex
	originals map[string]*OriginalFileInfo

	// 冻结标记（构建期写、运行期读）
	frozen bool

	// 冻结之后的快照
	snapshot ViewData
}

func NewInMemoryRegistry() Registry {
	return &memoryRegistry{
		originals: make(map[string]*OriginalFileInfo),
	}
}

// GetOriginal 在运行时读取原始文件信息
func (r *memoryRegistry) GetOriginal(path string) (*OriginalFileInfo, error) {
	if !r.frozen {
		return nil, oops.
			In("Registry.GetOriginal").
			With("path", path).
			Wrap(fmt.Errorf("registry not frozen"))
	}

	info, ok := lo.Find(r.snapshot.Originals, func(o *OriginalFileInfo) bool {
		return o.Path == path
	})
	if !ok {
		return nil, oops.
			In("Registry.GetOriginal").
			With("path", path).
			Wrap(ErrNotFound)
	}
	return info, nil
}

func (r *memoryRegistry) CountOriginals() int {
	if !r.frozen {
		return 0
	}
	return len(r.snapshot.Originals)
}

func (r *memoryRegistry) ListOriginals() []*OriginalFileInfo {
	if !r.frozen {
		return nil
	}
	return r.snapshot.Originals
}
func (r *memoryRegistry) ViewData() *ViewData {
	if !r.frozen {
		return nil
	}
	return &r.snapshot
}

// Freeze builds a read-only snapshot and disables further writes.
func (r *memoryRegistry) Freeze() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.frozen {
		return ErrFrozen
	}

	// collect all originals
	origList := lo.Values(r.originals)

	// sort for stable order
	sort.Slice(origList, func(i, j int) bool {
		return origList[i].Path < origList[j].Path
	})

	// write snapshot
	r.snapshot = ViewData{
		Originals: origList,
	}

	// clear original maps (enforce read-only)
	r.originals = nil
	r.frozen = true

	return nil
}

func (r *memoryRegistry) IsFrozen() bool {
	return r.frozen
}

// Writer 返回构建期写入接口
func (r *memoryRegistry) Writer() Writer {
	return &memoryWriter{r}
}
