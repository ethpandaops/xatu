package blockprint

import (
	"context"
	"encoding/json"
	"net/http"
)

type BlocksPerClientResponse struct {
	//nolint:tagliatelle // Defined by API.
	Uncertain uint64 `json:"Uncertain"`
	//nolint:tagliatelle // Defined by API.
	Lighthouse uint64 `json:"Lighthouse"`
	//nolint:tagliatelle // Defined by API.
	Lodestar uint64 `json:"Lodestar"`
	//nolint:tagliatelle // Defined by API.
	Nimbus uint64 `json:"Nimbus"`
	//nolint:tagliatelle // Defined by API.
	Other uint64 `json:"Other"`
	//nolint:tagliatelle // Defined by API.
	Prysm uint64 `json:"Prysm"`
	//nolint:tagliatelle // Defined by API.
	Teku uint64 `json:"Teku"`
	//nolint:tagliatelle // Defined by API.
	Grandine uint64 `json:"Grandine"`
}

type SyncStatusResponse struct {
	//nolint:tagliatelle // Defined by API.
	GreatestBlockSlot uint64 `json:"greatest_block_slot"`
	Synced            bool   `json:"synced"`
}

func (c *Client) BlocksPerClient(ctx context.Context, startEpoch, endEpoch string) (*BlocksPerClientResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.endpoint+"/blocks_per_client/"+startEpoch+"/"+endEpoch, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result BlocksPerClientResponse

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) SyncStatus(ctx context.Context) (*SyncStatusResponse, error) {
	data, err := c.get(ctx, "/sync/status")
	if err != nil {
		return nil, err
	}

	var result SyncStatusResponse

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
