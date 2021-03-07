package testdb

import (
	"net/http"

	"github.com/jarcoal/httpmock"
	"gitlab.com/thorchain/midgard/internal/fetch/chain/notinchain"
)

func MockThorNode(totalReserve int64, nodeAccounts []notinchain.NodeAccount) {
	thorNodeUrl := "http://thornode.com"
	notinchain.BaseURL = thorNodeUrl

	vaultData := notinchain.Network{TotalReserve: totalReserve}

	httpmock.RegisterResponder("GET", thorNodeUrl+"/nodes",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, nodeAccounts)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)

	httpmock.RegisterResponder("GET", thorNodeUrl+"/network",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, vaultData)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)
}
