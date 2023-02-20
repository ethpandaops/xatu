package service

type RawMessage struct {
	unmarshal func(interface{}) error
}

func (r *RawMessage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	r.unmarshal = unmarshal

	return nil
}

func (r *RawMessage) Unmarshal(v interface{}) error {
	return r.unmarshal(v)
}
