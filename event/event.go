// Package event provides the blockchain data in a structured way.
//
// Numeric values are 64 bits wide, instead of the conventional 256 bits used by
// most blockchains. Asset amounts are always represented with 8 decimals.
//
//	9 223 372 036 854 775 807  64-bit signed integer maximum
//	               00 000 000  decimals for fractions
//	   50 000 000 0·· ··· ···  500M Rune [ThorChain] total
//	    2 100 000 0·· ··· ···  21M BitCoin total
//	   20 000 000 0·· ··· ···  200M Ether total [Etheurem]
package event

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/tendermint/tendermint/libs/kv"
)

// Asset Labels
const (
	// Native asset on THORChain.
	Rune = "THOR.RUNE"
	// Asset on binance test net.
	rune67C = "BNB.RUNE-67C"
	// Asset on Binance main net.
	runeB1A = "BNB.RUNE-B1A"

	Bitcoin  = "BTC.BTC"
	Ethereum = "ETH.ETH"
	Binance  = "BNB.BNB"
)

func IsRune(asset []byte) bool {
	switch string(asset) {
	case Rune, rune67C, runeB1A:
		return true
	}
	return false
}

// CoinSep is the separator for coin lists.
var coinSep = []byte{',', ' '}

/*************************************************************/
/* Data models with Tendermint bindings in alphabetic order: */

// BUG(pascaldekloe): Duplicate keys in Tendermint transactions overwrite on another.

// Add defines the "add" event type.
type Add struct {
	TxID         []byte
	Chain        []byte
	FromAddr     []byte
	ToAddr       []byte
	Asset        []byte
	AmountE8     int64 // Asset quantity times 100 M
	Memo         []byte
	RuneAmountE8 int64 // Number of runes times 100 M

	Pool []byte
}

// LoadTendermint adopts the attributes.
func (e *Add) LoadTendermint(attrs []kv.Pair) error {
	*e = Add{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.TxID = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			b := attr.Value
			for len(b) != 0 {
				var asset []byte
				var amountE8 int64
				if i := bytes.Index(b, coinSep); i >= 0 {
					asset, amountE8, err = parseCoin(b[:i])
					b = b[i+len(coinSep):]
				} else {
					asset, amountE8, err = parseCoin(b)
					b = nil
				}
				if err != nil {
					return fmt.Errorf("malformed coin: %w", err)
				}

				if IsRune(asset) {
					e.RuneAmountE8 = amountE8
				} else {
					e.AmountE8 = amountE8
					e.Asset = asset
				}
			}
		case "memo":
			e.Memo = attr.Value

		case "pool":
			e.Pool = attr.Value

		default:
			log.Printf("unknown add event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Outbound defines the "fee" event type.
type Fee struct {
	InTxID     []byte
	Asset      []byte
	AmountE8   int64 // Asset quantity times 100 M
	PoolDeduct int64
}

// LoadTendermint adopts the attributes.
func (e *Fee) LoadTendermint(attrs []kv.Pair) error {
	*e = Fee{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "tx_id":
			e.InTxID = attr.Value
		case "coins":
			e.Asset, e.AmountE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coins: %w", err)
			}
		case "pool_deduct":
			e.PoolDeduct, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed pool_deduct: %w", err)
			}

		default:
			log.Printf("unknown fee event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Gas defines the "gas" event type.
type Gas struct {
	Asset        []byte
	AmountE8     int64 // Asset quantity times 100 M
	RuneAmountE8 int64 // Number of runes times 100 M
	TxCount      int64
}

// LoadTendermint adopts the attributes.
func (e *Gas) LoadTendermint(attrs []kv.Pair) error {
	*e = Gas{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "asset":
			e.Asset = attr.Value
		case "asset_amt":
			e.AmountE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed asset_amt: %w", err)
			}
		case "rune_amt":
			e.RuneAmountE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed rune_amt: %w", err)
			}
		case "transaction_count":
			e.TxCount, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed transaction_count: %w", err)
			}

		default:
			log.Printf("unknown gas event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Outbound defines the "outbound" event type.
type Outbound struct {
	TxID     []byte
	Chain    []byte
	FromAddr []byte
	ToAddr   []byte
	Asset    []byte
	AmountE8 int64 // Asset quantity times 100 M
	Memo     []byte

	InTxID []byte
}

// LoadTendermint adopts the attributes.
func (e *Outbound) LoadTendermint(attrs []kv.Pair) error {
	*e = Outbound{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.TxID = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AmountE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "in_tx_id":
			e.InTxID = attr.Value

		default:
			log.Printf("unknown outbound event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Pool defines the "pool" event type.
type Pool struct {
	Pool   []byte
	Status []byte
}

// LoadTendermint adopts the attributes.
func (e *Pool) LoadTendermint(attrs []kv.Pair) error {
	*e = Pool{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "pool":
			e.Pool = attr.Value
		case "pool_status":
			e.Status = attr.Value

		default:
			log.Printf("unknown pool event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Refund defines the "refund" event type.
type Refund struct {
	TxID     []byte
	Chain    []byte
	FromAddr []byte
	ToAddr   []byte
	Asset    []byte
	AmountE8 int64 // Asset quantity times 100 M
	Memo     []byte

	Code   int64
	Reason []byte
}

// LoadTendermint adopts the attributes.
func (e *Refund) LoadTendermint(attrs []kv.Pair) error {
	*e = Refund{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.TxID = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AmountE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "code":
			e.Code, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed code: %w", err)
			}
		case "reason":
			e.Reason = attr.Value

		default:
			log.Printf("unknown refund event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Reserve defines the "reserve" event type.
type Reserve struct {
	TxID     []byte
	Chain    []byte // redundant to asset
	FromAddr []byte
	ToAddr   []byte // may have multiple, separated by space
	Asset    []byte
	AmountE8 int64 // Asset quantity times 100 M
	Memo     []byte

	ContributorAddr     []byte
	ContributorAmountE8 int64 // Number of runes times 100 M
}

// LoadTendermint adopts the attributes.
func (e *Reserve) LoadTendermint(attrs []kv.Pair) error {
	*e = Reserve{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		// thornode: common.Tx
		case "id":
			e.TxID = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AmountE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "contributor_address":
			e.ContributorAddr = attr.Value
		case "amount":
			e.ContributorAmountE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed amount: %w", err)
			}

		default:
			log.Printf("unknown reserve event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Stake defines the "stake" event type.
type Stake struct {
	Pool          []byte
	StakeUnits    int64
	RuneAddr      []byte
	RuneAmountE8  int64 // Number of runes times 100 M
	AssetAmountE8 int64 // Asset quantity times 100 M
	RuneTxID      []byte
	AssetTxID     []byte
}

var txIDSuffix = []byte("_txid")

// LoadTendermint adopts the attributes.
func (e *Stake) LoadTendermint(attrs []kv.Pair) error {
	*e = Stake{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "pool":
			e.Pool = attr.Value
		case "stake_units":
			e.StakeUnits, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed stake_units: %w", err)
			}
		case "rune_address":
			e.RuneAddr = attr.Value
		case "rune_amount":
			e.RuneAmountE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed rune_amount: %w", err)
			}
		case "asset_amount":
			e.AssetAmountE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed asset_amount: %w", err)
			}

		case "THORChain_txid", "BNBChain_txid": // BNBChain for Binance test & main net
			e.RuneTxID = attr.Value

		default:
			switch {
			case bytes.HasSuffix(attr.Key, txIDSuffix):
				// attr.Key[:len(attr.Key)-len(txIDSuffix)] should equal .Pool chain
				e.AssetTxID = attr.Value
			default:
				log.Printf("unknown stake event attribute %q=%q", attr.Key, attr.Value)
			}
		}
	}

	if e.RuneTxID == nil {
		// omitted when equal
		e.RuneTxID = e.AssetTxID
	}

	return nil
}

// Swap defines the "swap" event type.
type Swap struct {
	TxID     []byte
	Chain    []byte
	FromAddr []byte
	ToAddr   []byte
	Asset    []byte
	AmountE8 int64 // Asset quantity times 100 M
	Memo     []byte

	Pool               []byte
	PriceTarget        int64
	TradeSlip          int64
	LiquidityFee       int64
	LiquidityFeeInRune int64
}

// LoadTendermint adopts the attributes.
func (e *Swap) LoadTendermint(attrs []kv.Pair) error {
	*e = Swap{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.TxID = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from_address":
			e.FromAddr = attr.Value
		case "to_address":
			e.ToAddr = attr.Value
		case "coins":
			e.Asset, e.AmountE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coins: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "pool":
			e.Pool = attr.Value
		case "price_target":
			e.PriceTarget, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed price_target: %w", err)
			}
		case "trade_slip":
			e.TradeSlip, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed trade_slip: %w", err)
			}
		case "liquidity_fee":
			e.LiquidityFee, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed liquidity_fee: %w", err)
			}
		case "liquidity_fee_in_rune":
			e.LiquidityFeeInRune, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed liquidity_fee_in_rune: %w", err)
			}

		default:
			log.Printf("unknown swap event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Unstake defines the "unstake" event type.
type Unstake struct {
	TxID     []byte
	Chain    []byte
	FromAddr []byte
	ToAddr   []byte
	Asset    []byte
	AmountE8 int64 // Asset quantity times 100 M
	Memo     []byte

	Pool        []byte
	StakeUnits  int64
	BasisPoints int64
	Asymmetry   float64 // lossy conversion
}

// LoadTendermint adopts the attributes.
func (e *Unstake) LoadTendermint(attrs []kv.Pair) error {
	*e = Unstake{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.TxID = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AmountE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "pool":
			e.Pool = attr.Value
		case "stake_units":
			e.StakeUnits, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed stake_units: %w", err)
			}
		case "basis_points":
			e.BasisPoints, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed basis_points: %w", err)
			}
		case "asymmetry":
			e.Asymmetry, err = strconv.ParseFloat(string(attr.Value), 64)
			if err != nil {
				return fmt.Errorf("malformed asymmetry: %w", err)
			}

		default:
			log.Printf("unknown unstake event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

var errNoSep = errors.New("separator not found")

func parseCoin(b []byte) (asset []byte, amountE8 int64, err error) {
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return nil, 0, errNoSep
	}
	asset = b[i+1:]
	amountE8, err = strconv.ParseInt(string(b[:i]), 10, 64)
	return
}
