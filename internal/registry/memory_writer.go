package registry

import (
	"fmt"

	"github.com/samber/oops"
)

type memoryWriter struct {
	reg *memoryRegistry
}

func (w *memoryWriter) RegisterOriginal(info *OriginalFileInfo) error {
	if info == nil {
		return oops.In("Registry.RegisterOriginal").
			Wrap(fmt.Errorf("info is nil"))
	}

	r := w.reg
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.frozen {
		return ErrFrozen
	}

	if info.Metrics == nil {
		info.Metrics = &Metrics{}
	}
	r.originals[info.Path] = info
	return nil
}

func (w *memoryWriter) AddVariant(path string, v *VariantFileInfo) error {
	r := w.reg
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.frozen {
		return ErrFrozen
	}

	if _, ok := r.originals[path]; !ok {
		return oops.In("Registry.AddVariant").
			With("path", path).
			Wrap(fmt.Errorf("original not registered"))
	}

	if v.Metrics == nil {
		v.Metrics = &Metrics{}
	}
	r.variants[path] = append(r.variants[path], v)
	return nil
}
