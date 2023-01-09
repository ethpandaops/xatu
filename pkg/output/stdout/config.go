package stdout

type Config struct {
	LoggingLevel string `yaml:"logging" default:"info"`
}

func (c *Config) Validate() error {
	return nil
}
