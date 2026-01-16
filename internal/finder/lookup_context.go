package finder

type LookupContext struct {
	AcceptEncoding string
	Path           string
}

func NewLookupContext(acceptEncoding string, path string) LookupContext {
	return LookupContext{
		acceptEncoding,
		path,
	}
}
