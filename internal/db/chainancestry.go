package db

var rootChainMap = map[string]FullyQualifiedChainId{
	"thorchain-v1": {"thorchain-testnet-v0", 1, "thorchain-testnet-v0"},
}

func RootChainIdOf(chainId string) FullyQualifiedChainId {
	ret := rootChainMap[chainId]
	for len(rootChainMap[ret.Name].Name) != 0 {
		ret = rootChainMap[ret.Name]
	}
	return ret
}
