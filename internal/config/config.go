package config

import appcfg "github.com/jayps/psql-transporter/internal/app/config"

const DefaultFile = appcfg.DefaultFile

type (
	Source = appcfg.Source
	Config = appcfg.Config
)

func EnsureExists(root string) (string, bool, error) { return appcfg.EnsureExists(root) }
func Load(path string) (Config, error)               { return appcfg.Load(path) }
func Save(path string, c Config) error               { return appcfg.Save(path, c) }
