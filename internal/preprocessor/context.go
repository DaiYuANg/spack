package preprocessor

type ctxKey string

const (
	ctxKeyHash ctxKey = "file_hash"
	ctxKeyMIME ctxKey = "file_mime"
)
