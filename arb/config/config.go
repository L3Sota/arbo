package config

type Config struct {
	PKey  string `split_words:"true"`
	PUser string `split_words:"true"`
}
