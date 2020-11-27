package timeseries

// This file can be deleted when Thornode stops sending invalid swap/stake events,
// and the testnet is deleted.
// https://gitlab.com/thorchain/thornode/-/merge_requests/1291/

// You can generate this list with the following command:
// select FORMAT('    "%s": true,', swap.tx) from swap_events as swap JOIN refund_events as refund ON swap.tx = refund.tx;
var swapBlacklist = map[string]bool{
	"0541BCD45307626C40B1EBB0EB78B5501683591BF9A17670883087E59BE5BB8A": true,
	"29C79E012BD5CCD395BAFC655DB2CE1B59B4410D84F576741713BFA6295B2614": true,
	"46B8DC657ACDEA7C12D92BB100D74764E0C8F70C0D49B5567B8596B7494684D6": true,
	"1ADD49D94C2B7DB6DD82C2A21C0CFA7CA515776C80F5ABC596FD8BD7434C98AE": true,
	"F2089AFA5A855BA6E425C1D0CC265E141DBE93BFDBA4D37374A415D88C9DA1A6": true,
	"828D446E1A32B7ABB33EE77C0E037F3CC894DABF626285DB2A4E504C57C38739": true,
	"550CE1D71E9B728C20B35F1858C3E8B4BB55329AF0FC33199FD0FBC76B32CBBF": true,
	"F2F0B1A108E59973CBA465D3DA4EBF660D436750249BA35B6168C6F293EF359D": true,
}

func isSwapBlacklisted(tx []byte) bool {
	_, ok := swapBlacklist[string(tx)]
	return ok
}
