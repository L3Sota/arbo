package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	PEnable bool   `split_words:"true"`
	PKey    string `split_words:"true"`
	PUser   string `split_words:"true"`

	KKey  string `split_words:"true"`
	KSec  string `split_words:"true"`
	KPass string `split_words:"true"`

	loaded bool
}

var c Config

func Load() *Config {
	if !c.loaded {
		envconfig.MustProcess("ARBO", &c)
		c.loaded = true
	}

	return &c
}
