package output

type RawMessage struct {
	unmarshal func(any) error
}

func (r *RawMessage) UnmarshalYAML(unmarshal func(any) error) error {
	r.unmarshal = unmarshal

	return nil
}

// Unmarshal decodes the captured block into v, leaving v untouched when there is no block.
func (r *RawMessage) Unmarshal(v any) error {
	if r == nil || r.unmarshal == nil {
		return nil
	}

	return r.unmarshal(v)
}
