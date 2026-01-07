package registry

type Metadata struct {
	registry Registry
}

func NewMetadata(registry Registry) *Metadata {
	return &Metadata{
		registry: registry,
	}
}
