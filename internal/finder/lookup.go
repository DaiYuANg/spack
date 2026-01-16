package finder

import (
	"log/slog"
	"os"
)

func (p *Finder) Lookup(ctx LookupOption) (*Result, error) {
	p.Info("LookupInternal", slog.Any("path", ctx.Path))
	objectInfo, err := p.registry.FindByPath(ctx.Path)
	if err != nil {
		return nil, err
	}
	p.Info("Lookup", slog.Any("path", objectInfo.FullPath))
	content, err := os.ReadFile(objectInfo.FullPath)
	if err != nil {
		p.Warn("Failed to read objectInfo file, fallback will be used", slog.String("path", objectInfo.Key), slog.String("err", err.Error()))
		return nil, err
	}
	// 如果读取失败，也可以 log 一下
	return &Result{
		Data:      content,
		MediaType: objectInfo.Mimetype,
	}, nil
}
