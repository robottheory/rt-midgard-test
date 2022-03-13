package db

var rootChainMap = map[string]FullyQualifiedChainId{
	"thorchain-v1": {"thorchain-testnet-v0", 1, "D4DF73AD98535DCD72BD0C9FE76B96CAF350C2FF517A61F77F5F89665A0593E7"},
}

func RootChainIdOf(chainId string) FullyQualifiedChainId {
	ret := rootChainMap[chainId]
	for len(rootChainMap[ret.Name].Name) != 0 {
		ret = rootChainMap[ret.Name]
	}
	return ret
}
