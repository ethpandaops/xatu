package stdout

type Config struct {
	LoggingLevel string `yaml:"logging" default:"info"`
	Pretty       bool   `yaml:"pretty" default:"false"`
}

func (c *Config) Validate() error {
	return nil
}
