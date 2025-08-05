package registry

// Entry RegistryEntry 表示静态文件的一次预处理结果
type Entry struct {
	RequestPath   string // /assets/a.png
	ActualPath    string // /tmp/v1/hash.webp
	MimeType      string // image/webp
	Preprocessors []string
	Version       string
}

// Registry 提供只写一次，多次读取的静态文件预处理映射
type Registry interface {
	Register(path string, entry *Entry) error
	Lookup(path string) (*Entry, bool)
	List() map[string]*Entry // 可选：调试用
}
