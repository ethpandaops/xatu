package geoip

type RawMessage struct {
	unmarshal func(any) error
}

func (r *RawMessage) UnmarshalYAML(unmarshal func(any) error) error {
	r.unmarshal = unmarshal

	return nil
}

func (r *RawMessage) Unmarshal(v any) error {
	return r.unmarshal(v)
}
