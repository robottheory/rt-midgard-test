// Package notinchain provides a temporary sollution for missing data in the blockchain.
// Remove the THOR node REST URL from the configuration once removed.
package notinchain

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BaseURL defines the REST root.
var BaseURL string

var Client http.Client

type NodeAccount struct {
	NodeAddr string `json:"node_address"`
	Status   string `json:"status"`
	Bond     int64  `json:"bond,string"`
}

func NodeAccountsLookup() ([]*NodeAccount, error) {
	resp, err := Client.Get(BaseURL + "/nodeaccounts")
	if err != nil {
		return nil, fmt.Errorf("node accounts unavailable from REST on %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("node accounts REST HTTP status %q, want 2xx", resp.Status)
	}
	var accounts []*NodeAccount
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, fmt.Errorf("node accounts irresolvable from REST on %w", err)
	}
	return accounts, nil
}
