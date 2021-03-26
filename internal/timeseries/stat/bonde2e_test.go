package stat_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestTotalBondE2E(t *testing.T) {
	testdb.InitTest(t)

	testdb.InsertBondEvent(t,
		testdb.FakeBond{Memo: "bonding", ToAddr: "node1", AssetE8: 1000, BondType: "bond_paid"},
	)
	testdb.InsertBondEvent(t,
		testdb.FakeBond{Memo: "unbonding", ToAddr: "node1", AssetE8: 300, BondType: "bond_returned"},
	)

	body := testdb.CallJSON(t, "http://localhost:8080/v2/stats")
	var jsonResult oapigen.StatsData
	testdb.MustUnmarshal(t, body, &jsonResult)

	assert.Equal(t, "700", jsonResult.TotalBond)
}
