package record

// Testnet started on 2021-07-10
const ChainIDTestnet202107 = "D6E12364E25D460C8D1155ADAD7CB827EE5D8D0B54B9609C928BF9EE9E23AC4C"

// ThorNode state and events diverged on testnet. We apply all these changes to be in sync with
// Thornode.
func loadTestnet202107Corrections(chainID string) {
	if chainID == ChainIDTestnet202107 {
	}
}
