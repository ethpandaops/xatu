package networks

type NetworkName string

type Network struct {
	Name NetworkName
	ID   uint64
}

var (
	NetworkNameNone    NetworkName = "none"
	NetworkNameUnknown NetworkName = "unknown"
	NetworkNameMainnet NetworkName = "mainnet"
	NetworkNameGoerli  NetworkName = "goerli"
	NetworkNameSepolia NetworkName = "sepolia"
)

var NetworkGenesisRoots = map[string]uint64{
	"0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95": 1,
	"0x043db0d9a83813551ee2f33450d23797757d430911a9320530ad8a0eabc43efb": 5,
	"0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078": 11155111,
}

var NetworkIDs = map[uint64]NetworkName{
	1:        NetworkNameMainnet,
	5:        NetworkNameGoerli,
	11155111: NetworkNameSepolia,
}

func DeriveFromGenesisRoot(genesisRoot string) *Network {
	if id, ok := NetworkGenesisRoots[genesisRoot]; ok {
		network := &Network{Name: NetworkNameUnknown, ID: id}
		if name, ok := NetworkIDs[id]; ok {
			network.Name = name
		}

		return network
	}

	return &Network{Name: NetworkNameUnknown, ID: 0}
}

func DeriveFromID(id uint64) *Network {
	network := &Network{Name: NetworkNameUnknown, ID: id}
	if name, ok := NetworkIDs[id]; ok {
		network.Name = name
	}

	return network
}
