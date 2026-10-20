// Package configdecode decodes a component's YAML config block over its default tags.
package configdecode

import "github.com/creasty/defaults"

// Unmarshaler is a deferred YAML decode, such as a config block captured as a RawMessage.
type Unmarshaler interface {
	Unmarshal(v any) error
}

// Decode sets conf's default tags and then decodes raw over them, so an explicit value, false or zero included, wins.
func Decode(raw Unmarshaler, conf any) error {
	if err := defaults.Set(conf); err != nil {
		return err
	}

	if raw == nil {
		return nil
	}

	return raw.Unmarshal(conf)
}
