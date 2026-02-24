package source

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// DiscoverTopics queries Kafka metadata and returns topic names that match
// at least one of the configured regex patterns, sorted alphabetically.
func DiscoverTopics(
	ctx context.Context,
	cfg *KafkaConfig,
) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil kafka config")
	}

	patterns, err := compileTopicPatterns(cfg.Topics)
	if err != nil {
		return nil, err
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
	}

	if cfg.TLS.Enabled {
		tlsCfg, tlsErr := cfg.TLS.Build()
		if tlsErr != nil {
			return nil, fmt.Errorf("building TLS config for topic discovery: %w", tlsErr)
		}

		opts = append(opts, kgo.DialTLSConfig(tlsCfg))
	}

	if cfg.SASLConfig != nil {
		mechanism, saslErr := franzSASLMechanism(cfg.SASLConfig)
		if saslErr != nil {
			return nil, fmt.Errorf("configuring SASL: %w", saslErr)
		}

		opts = append(opts, kgo.SASL(mechanism))
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client: %w", err)
	}
	defer cl.Close()

	req := kmsg.NewMetadataRequest()

	resp, err := req.RequestWith(ctx, cl)
	if err != nil {
		return nil, fmt.Errorf("fetching kafka metadata: %w", err)
	}

	return matchTopics(resp.Topics, patterns), nil
}

// compileTopicPatterns compiles the topic regex strings into regexp objects.
func compileTopicPatterns(topics []string) ([]*regexp.Regexp, error) {
	patterns := make([]*regexp.Regexp, 0, len(topics))

	for _, t := range topics {
		re, err := regexp.Compile(t)
		if err != nil {
			return nil, fmt.Errorf(
				"compiling topic pattern %q: %w", t, err,
			)
		}

		patterns = append(patterns, re)
	}

	return patterns, nil
}

// matchTopics filters Kafka metadata topics against compiled regex patterns
// and returns sorted, deduplicated topic names.
func matchTopics(
	topics []kmsg.MetadataResponseTopic,
	patterns []*regexp.Regexp,
) []string {
	seen := make(map[string]struct{}, len(topics))
	matched := make([]string, 0, len(topics))

	for _, t := range topics {
		name := ""
		if t.Topic != nil {
			name = *t.Topic
		}

		if name == "" {
			continue
		}

		if _, ok := seen[name]; ok {
			continue
		}

		for _, re := range patterns {
			if re.MatchString(name) {
				seen[name] = struct{}{}
				matched = append(matched, name)

				break
			}
		}
	}

	sort.Strings(matched)

	return matched
}
