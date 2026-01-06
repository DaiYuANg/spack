package preprocessor

import (
	"context"
	"errors"

	"github.com/daiyuang/spack/internal/registry"
	"github.com/daiyuang/spack/internal/scanner"
)

type VariantRunner struct {
	backend    scanner.Backend
	registry   registry.Registry
	processors []Processor
	writer     registry.Writer
}

func (r *VariantRunner) Run(ctx context.Context) error {
	if !r.registry.IsFrozen() {
		return errors.New("registry must be frozen")
	}

	for _, orig := range r.registry.ListOriginals() {
		obj, err := r.backend.Stat(orig.Path)
		if err != nil {
			return err
		}

		for _, p := range r.processors {
			if !p.Match(orig) {
				continue
			}

			plans := p.Plan(orig)
			for _, plan := range plans {
				if r.registry.HasVariants(orig.Path) {
					// 后续可升级为精确去重
				}

				if err := r.runOne(ctx, obj, orig, p, plan); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *VariantRunner) runOne(
	ctx context.Context,
	obj *scanner.ObjectInfo,
	orig *registry.OriginalFileInfo,
	p Processor,
	plan VariantPlan,
) error {

	//// 1. 创建目标 writer（fs / memory / temp）
	//w, commit, err := createVariantWriter(orig, plan)
	//if err != nil {
	//  return err
	//}
	//
	//// 2. 执行处理
	//size, err := p.Run(ctx, obj, orig, plan, w)
	//if err != nil {
	//  return err
	//}
	//
	//// 3. 提交存储并注册
	//path, err := commit()
	//if err != nil {
	//  return err
	//}
	//
	//return r.writer.AddVariant(orig.Path, &registry.VariantFileInfo{
	//  Path:        path,
	//  Ext:         plan.Ext,
	//  VariantType: plan.VariantType,
	//  Size:        size,
	//  Metrics:     &registry.Metrics{},
	//})
	return nil
}
