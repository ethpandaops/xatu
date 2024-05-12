package clmimicry

import (
	"context"
	"io"
	"net/http"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"gopkg.in/yaml.v2"
)

func (m *Mimicry) FetchConfigFromUpstream(ctx context.Context) (*params.BeaconChainConfig, error) {

	spec, err := m.ethereum.Node().FetchRawSpec(ctx)
	if err != nil {
		return nil, err
	}

	// Marshal the spec to a yaml string
	yamlSpec, err := yaml.Marshal(spec)
	if err != nil {
		return nil, err
	}

	config := &params.BeaconChainConfig{}

	out, err := params.UnmarshalConfig(yamlSpec, config)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func FetchBootnodeENRsFromURL(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the yaml data into the enrs struct
	var enrs []string
	err = yaml.Unmarshal(data, &enrs)
	if err != nil {
		return nil, err
	}

	return enrs, nil
}

func FetchDepositContractBlockFromURL(ctx context.Context, url string) (uint64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	// Unmarshal the yaml data into the block struct
	var block uint64

	err = yaml.Unmarshal(data, &block)
	if err != nil {
		return 0, err
	}

	return block, nil
}
