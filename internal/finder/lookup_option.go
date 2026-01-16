package finder

type LookupOption struct {
	AcceptEncoding string
	Path           string
}

func NewLookupContext(acceptEncoding string, path string) LookupOption {
	return LookupOption{
		AcceptEncoding: acceptEncoding,
		Path:           path,
	}
}
