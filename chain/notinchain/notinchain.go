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

type JailInfo struct {
	NodeAddr      string `json:"node_address"`
	ReleaseHeight int64  `json:"release_height,string"`
	Reason        string `json:"reason"`
}

type PublicKeys struct {
	Secp256k1 string `json:"secp256k1"`
	Ed25519   string `json:"ed25519"`
}

type NodeAccount struct {
	NodeAddr         string     `json:"node_address"`
	Status           string     `json:"status"`
	Bond             int64      `json:"bond,string"`
	PublicKeys       PublicKeys `json:"pub_key_set"`
	RequestedToLeave bool       `json:"requested_to_leave"`
	ForcedToLeave    bool       `json:"forced_to_leave"`
	LeaveHeight      int64      `json:"leave_height,string"`
	IpAddress        string     `json:"ip_address"`
	Version          string     `json:"version"`
	SlashPoints      int64      `json:"slash_points,string"`
	Jail             JailInfo   `json:"jail"`
	CurrentAward     int64      `json:"current_award,string"`
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

func NodeAccountLookup(addr string) (*NodeAccount, error) {
	resp, err := Client.Get(BaseURL + "/nodeaccount/" + addr)
	if err != nil {
		return nil, fmt.Errorf("node account unavailable from REST on %w", err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("node account REST HTTP status %q, want 2xx", resp.Status)
	}
	var account *NodeAccount
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("node account irresolvable from REST on %w", err)
	}
	return account, nil
}
