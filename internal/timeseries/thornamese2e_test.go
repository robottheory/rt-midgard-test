package timeseries_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/thorchain/midgard/internal/db/testdb"
	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

func TestTHORNamesE2E(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	setupLastBlock := int64(100)
	timeseries.SetLastHeightForTest(setupLastBlock)

	thor1 := "thor1xxxx"
	thor2 := "thor2xxxx"
	thor3 := "thor3xxxx"
	btc1 := "bc1xxxx"
	btc2 := "bc2xxxx"

	// setup a happy thorname
	blocks.NewBlock(t, "2000-01-01 00:00:00",
		testdb.THORName{
			Name:            "test1",
			Chain:           "THOR",
			Address:         thor1,
			Owner:           thor1,
			RegistrationFee: 10_00000000,
			FundAmount:      1_00000000,
			ExpireHeight:    123456,
		},
		testdb.THORName{
			Name:            "test1",
			Chain:           "BTC",
			Address:         btc1,
			RegistrationFee: 0,
			FundAmount:      0,
		},
	)
	var lookup oapigen.THORNameDetailsResponse
	body := testdb.CallJSON(t, "http://localhost:8080/v2/thorname/lookup/test1")
	testdb.MustUnmarshal(t, body, &lookup)

	require.Equal(t, 2, len(lookup.Entries))
	require.Equal(t, thor1, lookup.Owner)
	require.Equal(t, "123456", lookup.Expire)
	require.Equal(t, "BTC", lookup.Entries[0].Chain)
	require.Equal(t, btc1, lookup.Entries[0].Address)
	require.Equal(t, "THOR", lookup.Entries[1].Chain)
	require.Equal(t, thor1, lookup.Entries[1].Address)

	// Test chainging ownership of happy thorname
	blocks.NewBlock(t, "2000-01-01 00:02:00",
		testdb.THORName{
			Name:            "test1",
			Chain:           "THOR",
			Address:         thor2,
			Owner:           thor2,
			RegistrationFee: 0,
			FundAmount:      1_00000000,
			ExpireHeight:    1234567,
		},
		testdb.THORName{
			Name:            "test1",
			Chain:           "BTC",
			Address:         btc2,
			RegistrationFee: 0,
			FundAmount:      0,
		},
	)
	body = testdb.CallJSON(t, "http://localhost:8080/v2/thorname/lookup/test1")
	testdb.MustUnmarshal(t, body, &lookup)

	require.Equal(t, 2, len(lookup.Entries))
	require.Equal(t, thor2, lookup.Owner)
	require.Equal(t, "1234567", lookup.Expire)
	require.Equal(t, "BTC", lookup.Entries[0].Chain)
	require.Equal(t, btc2, lookup.Entries[0].Address)
	require.Equal(t, "THOR", lookup.Entries[1].Chain)
	require.Equal(t, thor2, lookup.Entries[1].Address)

	// check that an expired thorname doesn't show up
	blocks.NewBlock(t, "2000-01-01 00:03:00",
		testdb.THORName{
			Name:            "expired",
			Chain:           "THOR",
			Address:         thor1,
			Owner:           thor1,
			RegistrationFee: 10_00000000,
			FundAmount:      1_00000000,
			ExpireHeight:    1,
		},
	)
	testdb.CallFail(t, "http://localhost:8080/v2/thorname/lookup/expired", "not found")

	// Test reverse lookup, but first create a thorname, where the user is no longer the owner
	blocks.NewBlock(t, "2000-01-01 00:04:00",
		testdb.THORName{
			Name:            "test2",
			Chain:           "THOR",
			Address:         thor2,
			Owner:           thor2,
			RegistrationFee: 0,
			FundAmount:      1_00000000,
			ExpireHeight:    4000,
		},
	)
	blocks.NewBlock(t, "2000-01-01 00:05:00",
		testdb.THORName{
			Name:            "test2",
			Chain:           "THOR",
			Address:         thor3,
			Owner:           thor3,
			RegistrationFee: 0,
			FundAmount:      1_00000000,
			ExpireHeight:    4000,
		},
	)
	blocks.NewBlock(t, "2000-01-01 00:06:00",
		testdb.THORName{
			Name:            "test3",
			Chain:           "THOR",
			Address:         thor2,
			Owner:           thor2,
			RegistrationFee: 0,
			FundAmount:      1_00000000,
			ExpireHeight:    4000,
		},
	)
	var rlookup oapigen.ReverseTHORNameResponse
	testdb.CallFail(t, "http://localhost:8080/v2/thorname/rlookup/"+thor1, "not found")
	testdb.CallFail(t, "http://localhost:8080/v2/thorname/rlookup/"+btc1, "not found")

	body = testdb.CallJSON(t, "http://localhost:8080/v2/thorname/rlookup/"+thor2)
	testdb.MustUnmarshal(t, body, &rlookup)
	require.Equal(t, 2, len(rlookup))
	require.Equal(t, "test1", rlookup[0])
	require.Equal(t, "test3", rlookup[1])
}

func TestTHORNamesCaseInsensitive(t *testing.T) {
	blocks := testdb.InitTestBlocks(t)

	// setup a happy thorname
	blocks.NewBlock(t, "2000-01-01 00:00:00",
		testdb.THORName{
			Name:            "name1",
			Chain:           "THOR",
			Address:         "AddR1",
			Owner:           "ADDr1",
			RegistrationFee: 10_00000000,
			FundAmount:      1_00000000,
			ExpireHeight:    123456,
		},
	)

	var rlookup oapigen.ReverseTHORNameResponse

	body := testdb.CallJSON(t, "http://localhost:8080/v2/thorname/rlookup/aDdR1")
	testdb.MustUnmarshal(t, body, &rlookup)
	require.Equal(t, 1, len(rlookup))
	require.Equal(t, "name1", rlookup[0])
}
