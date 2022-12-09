package service

type Config struct {
	ServiceType Type `yaml:"type"`

	Config *RawMessage `yaml:"config"`
}
