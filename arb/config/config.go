package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	PEnable bool   `split_words:"true"`
	PKey    string `split_words:"true"`
	PUser   string `split_words:"true"`

	KKey  string `split_words:"true"`
	KSec  string `split_words:"true"`
	KPass string `split_words:"true"`

	HKey string `split_words:"true"`
	HSec string `split_words:"true"`

	GKey string `split_words:"true"`
	GSec string `split_words:"true"`

	CId  string `split_words:"true"`
	CSec string `split_words:"true"`

	ExecuteTrades bool `split_words:"true"`

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
