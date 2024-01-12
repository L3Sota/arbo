package config

type Config struct {
	PEnable bool   `split_words:"true"`
	PKey    string `split_words:"true"`
	PUser   string `split_words:"true"`
}
