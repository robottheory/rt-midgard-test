package timeseries_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"gitlab.com/thorchain/midgard/event"
	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/timeseries/testdb"
)

func intToBytes(n int64) []byte {
	return []byte(base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(int(n)))))
}

func insertOne(t *testing.T, n int64) {
	e := event.Swap{
		Tx:             intToBytes(n),
		Chain:          []byte("chain"),
		FromAddr:       intToBytes(n),
		ToAddr:         intToBytes(n),
		FromAsset:      []byte("BNB.BNB"),
		FromE8:         n,
		ToE8:           n,
		Memo:           intToBytes(n),
		Pool:           []byte("BNB.BNB"),
		ToE8Min:        n,
		TradeSlipBP:    n,
		LiqFeeE8:       n,
		LiqFeeInRuneE8: n,
	}
	height := n

	const q = `INSERT INTO swap_events (tx, chain, from_addr, to_addr, from_asset, from_E8, to_E8, memo, pool, to_E8_min, trade_slip_BP, liq_fee_E8, liq_fee_in_rune_E8, block_timestamp)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	result, err := db.Exec(
		q, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.FromAsset, e.FromE8, e.ToE8, e.Memo,
		e.Pool, e.ToE8Min, e.TradeSlipBP, e.LiqFeeE8, e.LiqFeeInRuneE8, height)
	if err != nil {
		t.Error("failed to insert:", err)
		return
	}
	k, err := result.RowsAffected()
	if err != nil {
		t.Error("failed to insert2: ", err)
		return
	}
	if k != 1 {
		t.Error("not one insert:", k)
	}
}

func valueStringIterator(argNum int) func() string {
	argCount := 0
	// return a string like ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) but from argcount
	return func() string {
		bb := bytes.Buffer{}
		bb.WriteString(" (")
		for k := 1; k <= argNum; k++ {
			argCount++

			if k != 1 {
				bb.WriteRune(',')
			}
			bb.WriteRune('$')
			bb.WriteString(strconv.Itoa(argCount))
		}
		bb.WriteString(")")
		return bb.String()
	}
}

func insertBatch(t *testing.T, from, to int64) {
	length := int(to - from)
	argNum := 14
	valueStrs := make([]string, 0, length)
	valueArgs := make([]interface{}, 0, argNum*length)
	insertIt := valueStringIterator(argNum)
	for n := from; n < to; n++ {
		e := event.Swap{
			Tx:             intToBytes(n),
			Chain:          []byte("chain"),
			FromAddr:       intToBytes(n),
			ToAddr:         intToBytes(n),
			FromAsset:      []byte("BNB.BNB"),
			FromE8:         n,
			ToE8:           n,
			Memo:           intToBytes(n),
			Pool:           []byte("BNB.BNB"),
			ToE8Min:        n,
			TradeSlipBP:    n,
			LiqFeeE8:       n,
			LiqFeeInRuneE8: n,
		}
		height := n
		valueStrs = append(valueStrs, insertIt())
		valueArgs = append(valueArgs, e.Tx, e.Chain, e.FromAddr, e.ToAddr, e.FromAsset, e.FromE8, e.ToE8, e.Memo,
			e.Pool, e.ToE8Min, e.TradeSlipBP, e.LiqFeeE8, e.LiqFeeInRuneE8, height)
	}
	q := fmt.Sprintf(
		`INSERT INTO swap_events (tx, chain, from_addr, to_addr, from_asset, from_E8, to_E8, memo, pool, to_E8_min, trade_slip_BP, liq_fee_E8, liq_fee_in_rune_E8, block_timestamp)
	VALUES %s`, strings.Join(valueStrs, ","))

	result, err := db.Exec(q, valueArgs...)
	if err != nil {
		t.Error("failed to insert:", err)
		return
	}
	k, err := result.RowsAffected()
	if err != nil {
		t.Error("failed to insert2: ", err)
		return
	}
	if int(k) != length {
		t.Error("not one insert:", k)
	}

}

func TestInsertOne(t *testing.T) {
	testdb.SetupTestDB(t)
	_, _ = db.Exec("DELETE FROM swap_events")
	insertOne(t, 0)
}

func TestInsertBatch(t *testing.T) {
	testdb.SetupTestDB(t)
	_, _ = db.Exec("DELETE FROM swap_events")
	insertBatch(t, 0, 4000)
}

func BenchmarkInsertOne(b *testing.B) {
	testdb.SetupTestDB(nil)
	_, _ = db.Exec("DELETE FROM swap_events")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		insertOne(nil, int64(i))
	}
}

// Max batch size we can use is ~4000 because there is a 64k limit on the
// sql argumentum size and we have 14 args per insert.
//
// The improvement is 73x:
//
// $ go test -run=NONE -bench Insert -v -p 1 ./...internal/timeseries...
// goos: linux
// goarch: amd64
// pkg: gitlab.com/thorchain/midgard/internal/timeseries
// BenchmarkInsertOne
// BenchmarkInsertOne-8                 682           1634591 ns/op
// BenchmarkInsertBatch
// BenchmarkInsertBatch-8             58502             22192 ns/op
// PASS
func BenchmarkInsertBatch(b *testing.B) {
	testdb.SetupTestDB(nil)
	_, _ = db.Exec("DELETE FROM swap_events")
	b.ResetTimer()
	batchSize := 4000
	for i := 0; i < b.N; i += batchSize {
		to := i + batchSize
		if b.N < to {
			to = b.N
		}
		insertBatch(nil, int64(i), int64(to))
	}
}
