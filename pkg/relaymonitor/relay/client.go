package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Config struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

var ErrValidatorNotRegistered = errors.New("validator not registered")

var ErrRateLimited = errors.New("rate limited")

// API endpoint constants
var (
	GetProposerPayloadEndpoint       = "get_proposer_payload"
	GetValidatorRegistrationEndpoint = "get_validator_registration"
)

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	if c.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type Client struct {
	baseURL     string
	name        string
	httpClient  *http.Client
	metrics     *Metrics
	networkName string // Add networkName to the Client struct
}

// NewClient creates a new relay client
func NewClient(namespace string, config Config, networkName string) (*Client, error) {
	_, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &Client{
		baseURL: config.URL,
		name:    config.Name,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		metrics:     GetMetrics(namespace),
		networkName: networkName,
	}, nil
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) URL() string {
	return c.baseURL
}

func (c *Client) GetBids(ctx context.Context, params url.Values) ([]*mevrelay.BidTrace, error) {
	reqURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	reqURL.Path = "/relay/v1/data/bidtraces/builder_blocks_received"
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start).Milliseconds()

	c.metrics.IncAPIRequests(c.name, GetBidsEndpoint, c.networkName)
	c.metrics.ObserveResponseTime(c.name, GetBidsEndpoint, c.networkName, float64(duration))

	if err != nil {
		c.metrics.IncAPIFailures(c.name, GetBidsEndpoint, c.networkName)

		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.metrics.IncAPIFailures(c.name, GetBidsEndpoint, c.networkName)

		var errorResp struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return nil, fmt.Errorf("failed to decode error response: %w", err)
		}

		return nil, fmt.Errorf("API error: %s (code: %d)", errorResp.Message, errorResp.Code)
	}

	//nolint:tagliatelle // Not our type
	var bids []*struct {
		Slot                 string `json:"slot"`
		ParentHash           string `json:"parent_hash"`
		BlockHash            string `json:"block_hash"`
		BuilderPubkey        string `json:"builder_pubkey"`
		ProposerPubkey       string `json:"proposer_pubkey"`
		ProposerFeeRecipient string `json:"proposer_fee_recipient"`
		GasLimit             string `json:"gas_limit"`
		GasUsed              string `json:"gas_used"`
		Value                string `json:"value"`
		NumTx                string `json:"num_tx"`
		BlockNumber          string `json:"block_number"`
		OptimisticSubmission bool   `json:"optimistic_submission"`
		Timestamp            string `json:"timestamp"`
		TimestampMs          string `json:"timestamp_ms"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&bids); err != nil {
		c.metrics.IncAPIFailures(c.name, GetBidsEndpoint, c.networkName)

		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	bidTraces := make([]*mevrelay.BidTrace, len(bids))

	for i, bid := range bids {
		slot, _ := strconv.ParseUint(bid.Slot, 10, 64)
		blockNumber, _ := strconv.ParseUint(bid.BlockNumber, 10, 64)
		gasLimit, _ := strconv.ParseUint(bid.GasLimit, 10, 64)
		gasUsed, _ := strconv.ParseUint(bid.GasUsed, 10, 64)
		numTx, _ := strconv.ParseUint(bid.NumTx, 10, 64)
		timestamp, _ := strconv.ParseInt(bid.Timestamp, 10, 64)
		timestampMs, _ := strconv.ParseInt(bid.TimestampMs, 10, 64)

		bidTraces[i] = &mevrelay.BidTrace{
			Slot:                 &wrapperspb.UInt64Value{Value: slot},
			ParentHash:           &wrapperspb.StringValue{Value: bid.ParentHash},
			BlockHash:            &wrapperspb.StringValue{Value: bid.BlockHash},
			BuilderPubkey:        &wrapperspb.StringValue{Value: bid.BuilderPubkey},
			ProposerPubkey:       &wrapperspb.StringValue{Value: bid.ProposerPubkey},
			ProposerFeeRecipient: &wrapperspb.StringValue{Value: bid.ProposerFeeRecipient},
			GasLimit:             &wrapperspb.UInt64Value{Value: gasLimit},
			GasUsed:              &wrapperspb.UInt64Value{Value: gasUsed},
			Value:                &wrapperspb.StringValue{Value: bid.Value},
			BlockNumber:          &wrapperspb.UInt64Value{Value: blockNumber},
			OptimisticSubmission: &wrapperspb.BoolValue{Value: bid.OptimisticSubmission},
			NumTx:                &wrapperspb.UInt64Value{Value: numTx},
			Timestamp:            &wrapperspb.Int64Value{Value: timestamp},
			TimestampMs:          &wrapperspb.Int64Value{Value: timestampMs},
		}
	}

	c.metrics.IncBidsReceived(c.name, c.networkName, len(bidTraces))

	return bidTraces, nil
}

func (c *Client) GetProposerPayloadDelivered(ctx context.Context, params url.Values) ([]*mevrelay.ProposerPayloadDelivered, error) {
	reqURL, err := url.Parse(c.baseURL + "/relay/v1/data/bidtraces/proposer_payload_delivered")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start).Milliseconds()

	c.metrics.IncAPIRequests(c.name, GetProposerPayloadEndpoint, c.networkName)
	c.metrics.ObserveResponseTime(c.name, GetProposerPayloadEndpoint, c.networkName, float64(duration))

	if err != nil {
		c.metrics.IncAPIFailures(c.name, GetProposerPayloadEndpoint, c.networkName)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.metrics.IncAPIFailures(c.name, GetProposerPayloadEndpoint, c.networkName)
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	//nolint:tagliatelle // Not our type
	var rawPayloads []*struct {
		Slot                 string `json:"slot"`
		ParentHash           string `json:"parent_hash"`
		BlockHash            string `json:"block_hash"`
		BuilderPubkey        string `json:"builder_pubkey"`
		ProposerPubkey       string `json:"proposer_pubkey"`
		ProposerFeeRecipient string `json:"proposer_fee_recipient"`
		GasLimit             string `json:"gas_limit"`
		GasUsed              string `json:"gas_used"`
		Value                string `json:"value"`
		BlockNumber          string `json:"block_number"`
		NumTx                string `json:"num_tx"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawPayloads); err != nil {
		c.metrics.IncAPIFailures(c.name, GetProposerPayloadEndpoint, c.networkName)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	payloads := make([]*mevrelay.ProposerPayloadDelivered, len(rawPayloads))

	for i, rawPayload := range rawPayloads {
		c.metrics.IncProposerPayloadDelivered(c.name, c.networkName, 1)

		slot, _ := strconv.ParseUint(rawPayload.Slot, 10, 64)
		gasLimit, _ := strconv.ParseUint(rawPayload.GasLimit, 10, 64)
		gasUsed, _ := strconv.ParseUint(rawPayload.GasUsed, 10, 64)
		blockNumber, _ := strconv.ParseUint(rawPayload.BlockNumber, 10, 64)
		numTx, _ := strconv.ParseUint(rawPayload.NumTx, 10, 64)

		payloads[i] = &mevrelay.ProposerPayloadDelivered{
			Slot:                 &wrapperspb.UInt64Value{Value: slot},
			ParentHash:           &wrapperspb.StringValue{Value: rawPayload.ParentHash},
			BlockHash:            &wrapperspb.StringValue{Value: rawPayload.BlockHash},
			BuilderPubkey:        &wrapperspb.StringValue{Value: rawPayload.BuilderPubkey},
			ProposerPubkey:       &wrapperspb.StringValue{Value: rawPayload.ProposerPubkey},
			ProposerFeeRecipient: &wrapperspb.StringValue{Value: rawPayload.ProposerFeeRecipient},
			GasLimit:             &wrapperspb.UInt64Value{Value: gasLimit},
			GasUsed:              &wrapperspb.UInt64Value{Value: gasUsed},
			Value:                &wrapperspb.StringValue{Value: rawPayload.Value},
			BlockNumber:          &wrapperspb.UInt64Value{Value: blockNumber},
			NumTx:                &wrapperspb.UInt64Value{Value: numTx},
		}
	}

	return payloads, nil
}

func (c *Client) GetValidatorRegistrations(ctx context.Context, pubkey string) (*mevrelay.ValidatorRegistration, error) {
	reqURL, err := url.Parse(c.baseURL + "/relay/v1/data/validator_registration")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	reqURL.RawQuery = url.Values{"pubkey": []string{pubkey}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start).Milliseconds()

	c.metrics.IncAPIRequests(c.name, GetValidatorRegistrationEndpoint, c.networkName)
	c.metrics.ObserveResponseTime(c.name, GetValidatorRegistrationEndpoint, c.networkName, float64(duration))

	if err != nil {
		c.metrics.IncAPIFailures(c.name, GetValidatorRegistrationEndpoint, c.networkName)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.metrics.IncAPIFailures(c.name, GetValidatorRegistrationEndpoint, c.networkName)

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}

		// Check for "no registration found"
		bodyBytes, errr := io.ReadAll(resp.Body)
		if errr != nil {
			return nil, fmt.Errorf("failed to read response body for non-200 status code: %w", errr)
		}

		if strings.Contains(string(bodyBytes), "no registration found") {
			return nil, ErrValidatorNotRegistered
		}

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rawRegistrations struct {
		Message struct {
			//nolint:tagliatelle // Not our type
			FeeRecipient string `json:"fee_recipient"`
			//nolint:tagliatelle // Not our type
			GasLimit  string `json:"gas_limit"`
			Timestamp string `json:"timestamp"`
			Pubkey    string `json:"pubkey"`
		} `json:"message"`
		Signature string `json:"signature"`
	}

	if errr := json.NewDecoder(resp.Body).Decode(&rawRegistrations); errr != nil {
		c.metrics.IncAPIFailures(c.name, GetValidatorRegistrationEndpoint, c.networkName)
		return nil, fmt.Errorf("failed to decode response: %w", errr)
	}

	gasLimit, err := strconv.ParseUint(rawRegistrations.Message.GasLimit, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas limit: %w", err)
	}

	timestamp, err := strconv.ParseUint(rawRegistrations.Message.Timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	registration := &mevrelay.ValidatorRegistration{
		Message: &mevrelay.ValidatorRegistrationMessage{
			FeeRecipient: &wrapperspb.StringValue{Value: rawRegistrations.Message.FeeRecipient},
			GasLimit:     &wrapperspb.UInt64Value{Value: gasLimit},
			Timestamp:    &wrapperspb.UInt64Value{Value: timestamp},
			Pubkey:       &wrapperspb.StringValue{Value: rawRegistrations.Message.Pubkey},
		},
		Signature: &wrapperspb.StringValue{Value: rawRegistrations.Signature},
	}

	c.metrics.IncValidatorRegistrationsReceived(c.name, c.networkName, 1)

	return registration, nil
}
