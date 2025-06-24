package scanner

import "context"

type MetaProcessor struct {
}

func (m MetaProcessor) Process(ctx context.Context, fullPath string, relPath string) error {
	//TODO implement me
	panic("implement me")
}

func (m MetaProcessor) Extensions() []string {
	//TODO implement me
	panic("implement me")
}
