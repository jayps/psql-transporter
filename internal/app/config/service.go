package config

type Loader interface {
	Load(path string) (Config, error)
}

// YAMLLoader satisfies Loader using the functions above.
type YAMLLoader struct{}

func NewYAMLLoader() *YAMLLoader { return &YAMLLoader{} }

func (YAMLLoader) Load(path string) (Config, error) { return Load(path) }
