package db

var rootChainMap = map[string]string{
	"771423E3B5F15BBA164BB54E0CD654FBC050494D98AC04A66C207494653A958D": "D4DF73AD98535DCD72BD0C9FE76B96CAF350C2FF517A61F77F5F89665A0593E7",
}

func RootChainIdOf(chainId string) string {
	for len(rootChainMap[chainId]) != 0 {
		chainId = rootChainMap[chainId]
	}
	return chainId
}
