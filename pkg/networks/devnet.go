package networks

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"golang.org/x/crypto/sha3"
	"gopkg.in/yaml.v3"
)

const maxUint64 = 18446744073709551615

// DevnetConfig represents the configuration for fetching network config from a URL.
type DevnetConfig struct {
	// URL is the base URL to the network config directory (e.g., https://raw.githubusercontent.com/ethpandaops/blob-devnets/master/network-configs/devnet-0)
	URL string `yaml:"url"`
}

// FetchedDevnetConfig contains all the computed network parameters from a devnet config.
type FetchedDevnetConfig struct {
	// ChainID is the network/chain ID from config.yaml DEPOSIT_CHAIN_ID
	ChainID uint64
	// GenesisHash is the execution layer genesis hash
	GenesisHash [32]byte
	// GenesisValidatorsRoot is the consensus layer GVR
	GenesisValidatorsRoot [32]byte
	// ForkVersions contains all fork versions from config.yaml
	ForkVersions []ForkVersionConfig
	// BootNodes contains ENR strings from bootstrap_nodes.txt (consensus layer)
	BootNodes []string
	// Enodes contains enode URLs from enodes.txt (execution layer)
	Enodes []string
	// ForkIDHash is the computed EIP-2124 fork ID hash (first 4 bytes of CRC32(genesis_hash))
	ForkIDHash [4]byte
	// ForkDigests contains computed fork digests for each fork version
	ForkDigests []ForkDigest
}

// ForkVersionConfig represents a fork version with its name.
type ForkVersionConfig struct {
	Name    string
	Version [4]byte
	Epoch   uint64
}

// ForkDigest represents a computed fork digest.
type ForkDigest struct {
	Name   string
	Digest [4]byte
}

// configYAML is parsed as a generic map to dynamically extract fork versions.
// This allows new forks to be supported without code changes.
type configYAML map[string]any

// getString returns a string value from the config map.
// For fork versions, YAML parses hex values as integers, so we convert them back.
func (c configYAML) getString(key string) string {
	if v, ok := c[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case int:
			// YAML parses hex like 0x10592455 as integers - convert back to hex
			return fmt.Sprintf("0x%x", val)
		case int64:
			return fmt.Sprintf("0x%x", val)
		}
	}

	return ""
}

// getUint64 returns a uint64 value from the config map.
func (c configYAML) getUint64(key string) uint64 {
	if v, ok := c[key]; ok {
		switch val := v.(type) {
		case int:
			if val >= 0 {
				return uint64(val)
			}
		case int64:
			if val >= 0 {
				return uint64(val)
			}
		case uint64:
			return val
		case float64:
			if val >= 0 {
				return uint64(val)
			}
		case string:
			if n, err := strconv.ParseUint(val, 10, 64); err == nil {
				return n
			}
		}
	}

	return 0
}

// Validate checks if the DevnetConfig is valid.
func (c *DevnetConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	return nil
}

// Fetch fetches and parses the network configuration from the URL.
func (c *DevnetConfig) Fetch(ctx context.Context) (*FetchedDevnetConfig, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	baseURL := strings.TrimSuffix(c.URL, "/")

	// Fetch all required files in parallel
	type fetchResult struct {
		name string
		data []byte
		err  error
	}

	// Required files - must exist
	requiredFiles := []string{
		"metadata/genesis.json",
		"metadata/config.yaml",
		"metadata/genesis_validators_root.txt",
		"metadata/bootstrap_nodes.txt",
	}

	// Optional files - ok if they don't exist
	optionalFiles := []string{
		"metadata/enodes.txt",
	}

	allFiles := make([]string, 0, len(requiredFiles)+len(optionalFiles))
	allFiles = append(allFiles, requiredFiles...)
	allFiles = append(allFiles, optionalFiles...)
	results := make(chan fetchResult, len(allFiles))

	for _, file := range allFiles {
		go func(fileName string) {
			data, err := c.fetchFile(ctx, fmt.Sprintf("%s/%s", baseURL, fileName))
			results <- fetchResult{name: fileName, data: data, err: err}
		}(file)
	}

	fetched := make(map[string][]byte, len(allFiles))
	optionalSet := make(map[string]bool, len(optionalFiles))

	for _, f := range optionalFiles {
		optionalSet[f] = true
	}

	for range allFiles {
		result := <-results
		if result.err != nil {
			// Only fail on required files
			if !optionalSet[result.name] {
				return nil, fmt.Errorf("failed to fetch %s: %w", result.name, result.err)
			}
			// Optional file not found - that's ok
			continue
		}

		fetched[result.name] = result.data
	}

	// Parse genesis.json to get genesis hash
	genesisHash, chainIDFromGenesis, err := c.parseGenesisJSON(fetched["metadata/genesis.json"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse genesis.json: %w", err)
	}

	// Parse config.yaml
	cfg, err := c.parseConfigYAML(fetched["metadata/config.yaml"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
	}

	// Use chain ID from config.yaml, fall back to genesis.json
	chainID := cfg.getUint64("DEPOSIT_CHAIN_ID")
	if chainID == 0 {
		chainID = chainIDFromGenesis
	}

	// Parse genesis validators root
	gvr, err := c.parseGenesisValidatorsRoot(fetched["metadata/genesis_validators_root.txt"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse genesis_validators_root.txt: %w", err)
	}

	// Parse bootstrap nodes (consensus layer ENRs)
	bootNodes := c.parseBootstrapNodes(fetched["metadata/bootstrap_nodes.txt"])

	// Parse enodes (execution layer) - optional
	var enodes []string
	if enodesData, ok := fetched["metadata/enodes.txt"]; ok {
		enodes = c.parseEnodes(enodesData)
	}

	// Build fork versions list
	forkVersions := c.buildForkVersions(cfg)

	// Compute fork ID hash (EIP-2124)
	forkIDHash := ComputeForkIDHash(genesisHash)

	// Compute fork digests for each active fork version
	forkDigests := c.computeForkDigests(forkVersions, gvr)

	return &FetchedDevnetConfig{
		ChainID:               chainID,
		GenesisHash:           genesisHash,
		GenesisValidatorsRoot: gvr,
		ForkVersions:          forkVersions,
		BootNodes:             bootNodes,
		Enodes:                enodes,
		ForkIDHash:            forkIDHash,
		ForkDigests:           forkDigests,
	}, nil
}

func (c *DevnetConfig) fetchFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *DevnetConfig) parseGenesisJSON(data []byte) (genesisHash [32]byte, chainID uint64, err error) {
	var genesis core.Genesis

	if err := genesis.UnmarshalJSON(data); err != nil {
		return [32]byte{}, 0, fmt.Errorf("failed to unmarshal genesis: %w", err)
	}

	// Compute genesis hash properly using go-ethereum's genesis block construction
	block := genesis.ToBlock()
	hash := block.Hash()

	copy(genesisHash[:], hash[:])

	if genesis.Config != nil {
		chainID = genesis.Config.ChainID.Uint64()
	}

	return genesisHash, chainID, nil
}

func (c *DevnetConfig) parseConfigYAML(data []byte) (*configYAML, error) {
	var cfg configYAML

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *DevnetConfig) parseGenesisValidatorsRoot(data []byte) ([32]byte, error) {
	var gvr [32]byte

	s := strings.TrimSpace(string(data))
	s = strings.TrimPrefix(s, "0x")

	if len(s) != 64 {
		return gvr, fmt.Errorf("invalid genesis validators root length: %d", len(s))
	}

	decoded, err := hex.DecodeString(s)
	if err != nil {
		return gvr, fmt.Errorf("invalid hex: %w", err)
	}

	copy(gvr[:], decoded)

	return gvr, nil
}

func (c *DevnetConfig) parseBootstrapNodes(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	nodes := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, "enr:") {
			nodes = append(nodes, line)
		}
	}

	return nodes
}

func (c *DevnetConfig) parseEnodes(data []byte) []string {
	lines := strings.Split(string(data), "\n")
	enodes := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, "enode://") {
			enodes = append(enodes, line)
		}
	}

	return enodes
}

func (c *DevnetConfig) buildForkVersions(cfg *configYAML) []ForkVersionConfig {
	versions := make([]ForkVersionConfig, 0)

	// Dynamically find all fork versions by looking for *_FORK_VERSION keys
	for key := range *cfg {
		if !strings.HasSuffix(key, "_FORK_VERSION") {
			continue
		}

		// Extract fork name (e.g., "GENESIS" from "GENESIS_FORK_VERSION")
		forkName := strings.TrimSuffix(key, "_FORK_VERSION")
		forkNameLower := strings.ToLower(forkName)

		versionStr := cfg.getString(key)
		if versionStr == "" {
			continue
		}

		version, err := parseForkVersion(versionStr)
		if err != nil {
			continue
		}

		// Get the corresponding epoch (genesis is always epoch 0)
		epoch := uint64(0)

		if forkNameLower != "genesis" {
			epochKey := forkName + "_FORK_EPOCH"
			epoch = cfg.getUint64(epochKey)

			// Skip disabled forks (epoch = max uint64)
			if epoch == maxUint64 {
				continue
			}
		}

		versions = append(versions, ForkVersionConfig{
			Name:    forkNameLower,
			Version: version,
			Epoch:   epoch,
		})
	}

	// Sort by epoch to ensure correct ordering
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Epoch < versions[j].Epoch
	})

	return versions
}

func (c *DevnetConfig) computeForkDigests(versions []ForkVersionConfig, gvr [32]byte) []ForkDigest {
	digests := make([]ForkDigest, 0, len(versions))

	for _, v := range versions {
		digest := ComputeForkDigest(v.Version, gvr)
		digests = append(digests, ForkDigest{
			Name:   v.Name,
			Digest: digest,
		})
	}

	return digests
}

// ComputeForkIDHash computes the EIP-2124 fork ID hash from a genesis hash.
// For a fresh network at genesis, FORK_HASH = CRC32(genesis_hash)
func ComputeForkIDHash(genesisHash [32]byte) [4]byte {
	crc := crc32.ChecksumIEEE(genesisHash[:])

	var hash [4]byte
	binary.BigEndian.PutUint32(hash[:], crc)

	return hash
}

// ComputeForkDigest computes the beacon chain fork digest.
// fork_digest = SHA256(fork_version || genesis_validators_root)[:4]
func ComputeForkDigest(forkVersion [4]byte, gvr [32]byte) [4]byte {
	// Concatenate fork_version and genesis_validators_root
	data := make([]byte, 36)
	copy(data[:4], forkVersion[:])
	copy(data[4:], gvr[:])

	// Compute SHA256 and take first 4 bytes
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)

	var digest [4]byte
	copy(digest[:], hasher.Sum(nil)[:4])

	return digest
}

func parseForkVersion(s string) ([4]byte, error) {
	var version [4]byte

	s = strings.TrimPrefix(s, "0x")

	if len(s) != 8 {
		return version, fmt.Errorf("invalid fork version length: %d", len(s))
	}

	decoded, err := hex.DecodeString(s)
	if err != nil {
		return version, err
	}

	copy(version[:], decoded)

	return version, nil
}

// ForkIDHashHex returns the fork ID hash as a hex string with 0x prefix.
func (f *FetchedDevnetConfig) ForkIDHashHex() string {
	return fmt.Sprintf("0x%x", f.ForkIDHash[:])
}

// ForkDigestHexes returns all fork digests as hex strings with 0x prefix.
func (f *FetchedDevnetConfig) ForkDigestHexes() []string {
	digests := make([]string, len(f.ForkDigests))

	for i, d := range f.ForkDigests {
		digests[i] = fmt.Sprintf("0x%x", d.Digest[:])
	}

	return digests
}

// CurrentForkDigestHex returns the most recent (highest epoch) fork digest as a hex string.
func (f *FetchedDevnetConfig) CurrentForkDigestHex() string {
	if len(f.ForkDigests) == 0 {
		return ""
	}

	// Return the last (most recent) fork digest
	return fmt.Sprintf("0x%x", f.ForkDigests[len(f.ForkDigests)-1].Digest[:])
}
