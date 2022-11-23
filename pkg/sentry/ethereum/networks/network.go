package networks

type NetworkName string

var (
	NetworkNameNone    NetworkName = "none"
	NetworkNameUnknown NetworkName = "unknown"
	NetworkNameMainnet NetworkName = "mainnet"
	NetworkNameSepolia NetworkName = "sepolia"
	NetworkNameGoerli  NetworkName = "goerli"
)

var NetworkGenesisRoots = map[string]NetworkName{
	"0x043db0d9a83813551ee2f33450d23797757d430911a9320530ad8a0eabc43efb": NetworkNameGoerli,
	"0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95": NetworkNameMainnet,
	"0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078": NetworkNameSepolia,
}

func DeriveNetworkName(genesisRoot string) NetworkName {
	if NetworkGenesisRoots[genesisRoot] != "" {
		return NetworkGenesisRoots[genesisRoot]
	}

	return NetworkNameUnknown
}
