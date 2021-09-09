package testdb

import (
	"net/http"

	"github.com/jarcoal/httpmock"
	"gitlab.com/thorchain/midgard/internal/fetch/notinchain"
)

const thorNodeUrl = "http://thornode.com"

// Starts Thornode HTTP mock with some simiple / empty results.
func StartMockThornode() (deactivateCallback func()) {
	notinchain.BaseURL = thorNodeUrl

	RegisterThornodeNodes([]notinchain.NodeAccount{})
	RegisterThornodeReserve(0)

	httpmock.Activate()
	return func() {
		httpmock.DeactivateAndReset()
	}
}

func RegisterThornodeNodes(nodeAccounts []notinchain.NodeAccount) {
	httpmock.RegisterResponder("GET", thorNodeUrl+"/nodes",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(200, nodeAccounts)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)
}

func RegisterThornodeReserve(totalReserve int64) {
	httpmock.RegisterResponder("GET", thorNodeUrl+"/network",
		func(req *http.Request) (*http.Response, error) {
			vaultData := notinchain.Network{TotalReserve: totalReserve}
			resp, err := httpmock.NewJsonResponse(200, vaultData)
			if err != nil {
				return httpmock.NewStringResponse(500, ""), nil
			}
			return resp, nil
		},
	)
}
