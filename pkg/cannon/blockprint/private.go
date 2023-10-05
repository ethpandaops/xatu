package blockprint

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

type ProposersBlocksResponse struct {
	//nolint:tagliatelle // Defined by API.
	ProposerIndex uint64 `json:"proposer_index"`
	Slot          uint64 `json:"slot"`
	//nolint:tagliatelle // Defined by API.
	BestGuessSingle ClientName `json:"best_guess_single"`
	//nolint:tagliatelle // Defined by API.
	BestGuessMulti string `json:"best_guess_multi"`
	//nolint:tagliatelle // Defined by API.
	ProbabilityMap *ProbabilityMap `json:"probability_map"`
}

func (c *Client) BlocksRange(ctx context.Context, startSlot phase0.Slot, endSlot ...phase0.Slot) ([]*ProposersBlocksResponse, error) {
	var path string
	if len(endSlot) > 0 {
		path = fmt.Sprintf("/blocks/%d/%d", startSlot, endSlot[0])
	} else {
		path = fmt.Sprintf("/blocks/%d", startSlot)
	}

	data, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result []*ProposersBlocksResponse

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
