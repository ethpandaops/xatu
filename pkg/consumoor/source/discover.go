package source

import (
	"context"
	"crypto/tls"
	"fmt"
	"regexp"
	"sort"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// DiscoverTopics connects to Kafka, lists all topics, and returns
// those whose names match any of the regex patterns in cfg.Topics.
// The returned slice is sorted lexicographically.
func DiscoverTopics(ctx context.Context, cfg *KafkaConfig) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil kafka config")
	}

	compiled := make([]*regexp.Regexp, 0, len(cfg.Topics))
	for _, pattern := range cfg.Topics {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("compiling topic pattern %q: %w", pattern, err)
		}

		compiled = append(compiled, re)
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
	}

	if cfg.TLS {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{
			MinVersion: tls.VersionTLS12,
		}))
	}

	if cfg.SASLConfig != nil {
		mechanism, err := franzSASLMechanism(cfg.SASLConfig)
		if err != nil {
			return nil, fmt.Errorf("configuring SASL for topic discovery: %w", err)
		}

		opts = append(opts, kgo.SASL(mechanism))
	}

	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating kafka admin client: %w", err)
	}
	defer cl.Close()

	admin := kadm.NewClient(cl)
	defer admin.Close()

	details, err := admin.ListTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing kafka topics: %w", err)
	}

	allNames := details.Names()

	matched := make([]string, 0, len(allNames))

	for _, name := range allNames {
		for _, re := range compiled {
			if re.MatchString(name) {
				matched = append(matched, name)

				break
			}
		}
	}

	sort.Strings(matched)

	return matched, nil
}
