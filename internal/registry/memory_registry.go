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
	variants  map[string][]*VariantFileInfo

	// 冻结标记（构建期写、运行期读）
	frozen bool

	// 冻结之后的快照
	snapshot ViewData
}

func NewInMemoryRegistry() Registry {
	return &memoryRegistry{
		originals: make(map[string]*OriginalFileInfo),
		variants:  make(map[string][]*VariantFileInfo),
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

// GetVariants 在运行时读取变体列表
func (r *memoryRegistry) GetVariants(path string) ([]*VariantFileInfo, error) {
	if !r.frozen {
		return nil, oops.
			In("Registry.GetVariants").
			With("path", path).
			Wrap(fmt.Errorf("registry not frozen"))
	}

	// filter snapshot
	var out []*VariantFileInfo
	for _, v := range r.snapshot.Variants {
		if v.OriginalPath == path {
			out = append(out, v.VariantFileInfo)
		}
	}
	if len(out) == 0 {
		return nil, oops.
			In("Registry.GetVariants").
			With("path", path).
			Wrap(ErrNotFound)
	}
	return out, nil
}

func (r *memoryRegistry) HasVariants(path string) bool {
	if !r.frozen {
		return false
	}
	for _, v := range r.snapshot.Variants {
		if v.OriginalPath == path {
			return true
		}
	}
	return false
}

func (r *memoryRegistry) CountOriginals() int {
	if !r.frozen {
		return 0
	}
	return len(r.snapshot.Originals)
}

func (r *memoryRegistry) CountVariants(path string) int {
	if !r.frozen {
		return 0
	}
	return countVariantsOf(r.snapshot.Variants, path)
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

	// flatten variants into snapshot view
	flatVariants := lo.FlatMap(origList, func(o *OriginalFileInfo, _ int) []*VariantView {
		return lo.Map(r.variants[o.Path], func(v *VariantFileInfo, _ int) *VariantView {
			return &VariantView{
				OriginalPath:    o.Path,
				VariantFileInfo: v,
			}
		})
	})

	// sort flat variants for stable output
	sort.Slice(flatVariants, func(i, j int) bool {
		return flatVariants[i].OriginalPath < flatVariants[j].OriginalPath
	})

	// write snapshot
	r.snapshot = ViewData{
		Originals: origList,
		Variants:  flatVariants,
	}

	// clear original maps (enforce read-only)
	r.originals = nil
	r.variants = nil
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

func countVariantsOf(snapshot []*VariantView, path string) int {
	return len(lo.Filter(snapshot, func(v *VariantView, _ int) bool {
		return v.OriginalPath == path
	}))
}
