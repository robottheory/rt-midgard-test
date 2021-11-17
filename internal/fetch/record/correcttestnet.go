package record

// Testnet started on 2021-11-06
const ChainIDTestnet20211106 = "D4DF73AD98535DCD72BD0C9FE76B96CAF350C2FF517A61F77F5F89665A0593E7"

// ThorNode state and events diverged on testnet. We apply all these changes to be in sync with
// Thornode.
func loadTestnet202107Corrections(chainID string) {
	if chainID == ChainIDTestnet20211106 {
	}
}
