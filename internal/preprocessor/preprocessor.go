package preprocessor

type Preprocessor interface {
	Name() string
	Order() int
	CanProcess(path string, mime string) bool
	Process(path string) error
}
