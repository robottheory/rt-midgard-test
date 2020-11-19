package testdb

import (
	"github.com/jarcoal/httpmock"
	"gitlab.com/thorchain/midgard/chain/notinchain"
	"net/http"
)

func MockThorNode(totalReserve int64, nodeAccounts []notinchain.NodeAccount) {
	thorNodeUrl := "http://thornode.com"
	notinchain.BaseURL = thorNodeUrl

	vaultData := notinchain.VaultData{TotalReserve: totalReserve}

	httpmock.RegisterResponder("GET", thorNodeUrl+"/nodeaccounts",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, nodeAccounts)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)

	httpmock.RegisterResponder("GET", thorNodeUrl+"/vault",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, vaultData)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)
}
