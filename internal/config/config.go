package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultFile = "psql-transporter.yaml"

type Source struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	DBName    string `yaml:"dbname"`
	SSLMode   string `yaml:"sslmode"`
	Protected bool   `yaml:"protected"`
}

type Config struct {
	Sources []Source `yaml:"sources"`
}

func EnsureExists(root string) (string, bool, error) {
	cfgPath := filepath.Join(root, DefaultFile)
	_, err := os.Stat(cfgPath)
	if errors.Is(err, os.ErrNotExist) {
		def := Config{
			Sources: []Source{
				{
					Name: "example",
					Host: "127.0.0.1", Port: 5432,
					User: "postgres", Password: "postgres",
					DBName: "app_db", SSLMode: "disable",
					Protected: true,
				},
			},
		}
		if err := Save(cfgPath, def); err != nil {
			return cfgPath, false, err
		}
		return cfgPath, true, nil
	}
	return cfgPath, false, err
}

func Load(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if len(c.Sources) == 0 {
		return c, errors.New("config has no sources")
	}
	return c, nil
}

func Save(path string, c Config) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
