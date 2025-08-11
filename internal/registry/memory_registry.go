package registry

import (
	"path/filepath"
	"sync"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type InMemoryRegistry struct {
	muOrig    sync.RWMutex
	originals cmap.ConcurrentMap[string, *OriginalFileInfo]

	muVar    sync.RWMutex
	variants cmap.ConcurrentMap[string, []*VariantFileInfo]
	logger   *zap.SugaredLogger
}

func NewInMemoryRegistry(logger *zap.SugaredLogger) (*InMemoryRegistry, error) {
	originals := cmap.New[*OriginalFileInfo]()
	variants := cmap.New[[]*VariantFileInfo]()
	return &InMemoryRegistry{
		variants:  variants,
		logger:    logger,
		originals: originals,
	}, nil
}

func (r *InMemoryRegistry) RegisterOriginal(info *OriginalFileInfo) error {
	r.muOrig.Lock()
	defer r.muOrig.Unlock()
	r.originals.Set(info.Path, info)
	return nil
}
func (r *InMemoryRegistry) GetOriginal(path string) (*OriginalFileInfo, error) {
	r.muOrig.RLock()
	defer r.muOrig.RUnlock()
	info, ok := r.originals.Get(path)
	if !ok {
		return nil, ErrNotFound
	}
	return info, nil
}

// AddVariant 增加一个变体文件
func (r *InMemoryRegistry) AddVariant(originalPath string, variant *VariantFileInfo) {
	r.muVar.Lock()
	defer r.muVar.Unlock()
	oldSlice, exists := r.variants.Get(originalPath)
	if !exists {
		oldSlice = []*VariantFileInfo{}
	}

	newSlice := append(oldSlice, variant)
	r.variants.Set(originalPath, newSlice)
}
func (r *InMemoryRegistry) GetVariants(originalPath string) ([]*VariantFileInfo, error) {
	val, exists := r.variants.Get(originalPath)
	if !exists {
		return []*VariantFileInfo{}, nil
	}
	return val, nil
}

func (r *InMemoryRegistry) HasVariants(originalPath string) bool {
	val, exists := r.variants.Get(originalPath)
	if !exists {
		return false
	}
	return len(val) > 0
}

func (r *InMemoryRegistry) ExistsOriginal(path string) bool {
	_, exists := r.originals.Get(path)
	return exists
}

func (r *InMemoryRegistry) ListOriginals() []string {
	r.muOrig.RLock()
	defer r.muOrig.RUnlock()

	var keys []string
	r.originals.IterCb(func(key string, _ *OriginalFileInfo) {
		keys = append(keys, key)
	})
	return keys
}

// CountOriginals 返回当前注册的原始文件数量
func (r *InMemoryRegistry) CountOriginals() int {
	r.muOrig.RLock()
	defer r.muOrig.RUnlock()
	return r.originals.Count()
}

// CountVariants 返回某个原始文件的变体数量
func (r *InMemoryRegistry) CountVariants(originalPath string) int {
	val, exists := r.variants.Get(originalPath)
	// 用 lo.Optional 封装存在性判断，返回长度或 0
	return lo.Ternary(exists, len(val), 0)
}

// BatchAddVariants 批量添加多个变体，减少锁竞争
func (r *InMemoryRegistry) BatchAddVariants(originalPath string, variants []*VariantFileInfo) {
	r.muVar.Lock()
	defer r.muVar.Unlock()

	oldSlice, exists := r.variants.Get(originalPath)
	if !exists {
		oldSlice = []*VariantFileInfo{}
	}
	newSlice := append(oldSlice, variants...)
	r.variants.Set(originalPath, newSlice)
}

func (r *InMemoryRegistry) ListOriginal() []*OriginalFileInfo {
	r.muOrig.RLock()
	defer r.muOrig.RUnlock()

	var infos []*OriginalFileInfo
	r.originals.IterCb(func(_ string, info *OriginalFileInfo) {
		infos = append(infos, info)
	})
	return infos
}

func (r *InMemoryRegistry) ViewData() *ViewData {
	// 获取所有原始文件
	originals := r.ListOriginal()

	// 整理变体
	var variants []*VariantView
	r.muVar.RLock()
	defer r.muVar.RUnlock()
	r.variants.IterCb(func(orig string, vs []*VariantFileInfo) {
		for _, v := range vs {
			variants = append(variants, &VariantView{
				OriginalPath:    orig,
				VariantFileInfo: v,
			})
		}
	})

	lo.ForEach(originals, func(info *OriginalFileInfo, _ int) {
		info.Path = filepath.Base(info.Path)
	})

	lo.ForEach(variants, func(item *VariantView, index int) {
		item.Path = filepath.Base(item.Path)
		item.OriginalPath = filepath.Base(item.OriginalPath)
	})

	return &ViewData{
		Originals: originals,
		Variants:  variants,
	}
}
