package service

type Config struct {
	ServiceType ServiceType `yaml:"type"`

	Config *RawMessage `yaml:"config"`
}
