// Package configenv decodes YAML config files, expanding ${VAR} and
// ${VAR:-default} placeholders in their scalar values from the environment.
package configenv

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)

// kafkaTopicVariables are resolved per event by the kafka output's topicPattern, not from the environment.
var kafkaTopicVariables = []string{"${EVENT_NAME}", "${event-name}"}

// Decode parses data, expands placeholders in scalar values (not mapping
// keys) and decodes the result into out. A ${VAR} without a default whose
// VAR is unset or empty is an error. A ${VAR:-default} uses the default when
// VAR is unset or empty. Data without "${" decodes exactly as yaml.Unmarshal.
func Decode(data []byte, out any) error {
	if !bytes.Contains(data, []byte("${")) {
		return yaml.Unmarshal(data, out)
	}

	doc, err := expand(data)
	if err != nil {
		return err
	}

	return doc.Decode(out)
}

// DecodeStrict is Decode, but a mapping key without a matching field in out
// is an error, as with a yaml.Decoder that has KnownFields set.
func DecodeStrict(data []byte, out any) error {
	if bytes.Contains(data, []byte("${")) {
		doc, err := expand(data)
		if err != nil {
			return err
		}

		// yaml.v3 only checks known fields when decoding from a stream,
		// so the expanded tree is encoded again for the strict decoder.
		if data, err = yaml.Marshal(doc); err != nil {
			return err
		}
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	return dec.Decode(out)
}

func expand(data []byte) (*yaml.Node, error) {
	var doc yaml.Node

	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var missing []string

	expandNode(&doc, &missing)

	if len(missing) > 0 {
		return nil, fmt.Errorf("config references unset or empty environment variables: %s", strings.Join(missing, ", "))
	}

	return &doc, nil
}

func expandNode(n *yaml.Node, missing *[]string) {
	switch n.Kind {
	case yaml.ScalarNode:
		expanded := expandEnvVars(n.Value, missing)
		if expanded == n.Value {
			return
		}

		n.Value = expanded

		// The parser tagged the placeholder !!str; clearing the tag lets an
		// unquoted value resolve to int, bool, etc. as if written in the file.
		// A value that would resolve to null stays a string instead of
		// silently decoding as empty.
		if n.Style == 0 {
			n.Tag = ""

			if n.ShortTag() == "!!null" {
				n.Tag = "!!str"
			}
		}
	case yaml.MappingNode:
		for i := 1; i < len(n.Content); i += 2 {
			expandNode(n.Content[i], missing)
		}
	case yaml.DocumentNode, yaml.SequenceNode:
		for _, c := range n.Content {
			expandNode(c, missing)
		}
	case yaml.AliasNode:
		// An alias decodes its anchor's node, which is expanded where it is defined.
	}
}

func expandEnvVars(value string, missing *[]string) string {
	return envVarPattern.ReplaceAllStringFunc(value, func(match string) string {
		if slices.Contains(kafkaTopicVariables, match) {
			return match
		}

		parts := envVarPattern.FindStringSubmatchIndex(match)
		name := match[parts[2]:parts[3]]

		if v := os.Getenv(name); v != "" {
			return v
		}

		if parts[4] >= 0 {
			return match[parts[4]:parts[5]]
		}

		if !slices.Contains(*missing, name) {
			*missing = append(*missing, name)
		}

		return ""
	})
}
