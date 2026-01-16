package finder

import (
	"log/slog"
	"os"
)

func (p *Finder) Lookup(ctx LookupOption) (*Result, error) {
	p.Info("Lookup", slog.Any("path", ctx.Path))
	original, err := p.registry.GetOriginal(ctx.Path)
	if err != nil {
		return nil, err
	}
	p.Info("Lookup", slog.Any("path", original.FullPath))
	content, err := os.ReadFile(original.FullPath)
	if err != nil {
		p.Warn("Failed to read original file, fallback will be used", slog.String("path", original.Path), slog.StringValue(err.Error()))
		return nil, err
	}
	// 如果读取失败，也可以 log 一下
	return &Result{
		Data:      content,
		MediaType: original.Mimetype,
	}, nil
}
