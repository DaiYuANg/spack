package scanner

import "go.uber.org/fx"

func asProcessor(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(FileProcessor)),
		fx.ResultTags(`group:"processor"`),
	)
}
