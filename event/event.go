// Package event provides the blockchain data in a structured way.
package event

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/tendermint/tendermint/libs/kv"
)

/*************************************************************/
/* Data models with Tendermint bindings in alphabetic order: */

// BUG(pascaldekloe): Duplicate keys in Tendermint transactions overwrite on another.

// TODO(pascaldekloe): Reuse byte slices with LoadTendermint.

// Add defines the "add" event type.
type Add struct {
	TxID       []byte
	Chain      []byte
	FromAddr   []byte
	ToAddr     []byte
	CoinAsset  []byte
	CoinAmount int64
	Memo       []byte
	Pool       []byte
}

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
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
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
	CoinAsset  []byte
	CoinAmount int64
	PoolDeduct int64
}

func (e *Fee) LoadTendermint(attrs []kv.Pair) error {
	*e = Fee{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "tx_id":
			e.InTxID = attr.Value
		case "pool_deduct":
			e.PoolDeduct, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed pool_deduct: %w", err)
			}
		case "coins":
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coins: %w", err)
			}
		default:
			log.Printf("unknown fee event attribute %q=%q", attr.Key, attr.Value)
		}
	}
	return nil
}

// Message defines the "message" event type.
type Message struct {
	Action []byte
}

func (e *Message) LoadTendermint(attrs []kv.Pair) error {
	*e = Message{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "action":
			e.Action = attr.Value
		default:
			log.Printf("unknown message event attribute %q=%q", attr.Key, attr.Value)
		}
	}
	return nil
}

// Outbound defines the "outbound" event type.
type Outbound struct {
	InTxID     []byte
	TxID       []byte
	Chain      []byte
	FromAddr   []byte
	ToAddr     []byte
	CoinAsset  []byte
	CoinAmount int64
	Memo       []byte
}

func (e *Outbound) LoadTendermint(attrs []kv.Pair) error {
	*e = Outbound{}
	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "in_tx_id":
			e.InTxID = attr.Value
		case "id":
			e.TxID = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value
		default:
			log.Printf("unknown outbound event attribute %q=%q", attr.Key, attr.Value)
		}
	}
	return nil
}

// Pool defines the "pool" event type.
type Pool struct {
	TxID       []byte
	Chain      []byte
	FromAddr   []byte
	ToAddr     []byte
	CoinAsset  []byte
	CoinAmount int64
	Memo       []byte
	Pool       []byte
	Status     []byte
}

func (e *Pool) LoadTendermint(attrs []kv.Pair) error {
	*e = Pool{}
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
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value
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
	TxID       []byte
	Chain      []byte
	FromAddr   []byte
	ToAddr     []byte
	CoinAsset  []byte
	CoinAmount int64
	Memo       []byte
	Code       int64
	Reason     []byte
}

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
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value
		case "code":
			e.Code, err = strconv.ParseInt(string(attr.Value), 10, 64)
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
	TxID        []byte
	Chain       []byte
	FromAddr    []byte
	ToAddr      []byte
	CoinAsset   []byte
	CoinAmount  int64
	Memo        []byte
	ContribAddr []byte
	Amount      int64
}

func (e *Reserve) LoadTendermint(attrs []kv.Pair) error {
	*e = Reserve{}

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
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value
		case "contributor_address":
			e.ContribAddr = attr.Value
		case "amount":
			e.Amount, err = strconv.ParseInt(string(attr.Value), 10, 64)
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
	Chain         []byte
	Memo          []byte
	Pool          []byte
	StakeUnits    int64
	RuneAddr      []byte
	RuneAmount    int64
	AssetAmount   int64
	TxIDsPerChain map[string][][]byte
}

var txIDSuffix = []byte("_txid")

func (e *Stake) LoadTendermint(attrs []kv.Pair) error {
	*e = Stake{}
	e.TxIDsPerChain = make(map[string][][]byte)

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "chain":
			e.Chain = attr.Value
		case "memo":
			e.Memo = attr.Value
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
			e.RuneAmount, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed rune_amount: %w", err)
			}
		case "asset_amount":
			e.AssetAmount, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed asset_amount: %w", err)
			}
		default:
			switch {
			case bytes.HasSuffix(attr.Key, txIDSuffix):
				chain := string(attr.Value)
				e.TxIDsPerChain[chain] = append(e.TxIDsPerChain[chain], attr.Key[:len(attr.Key)-len(txIDSuffix)])
			default:
				log.Printf("unknown stake event attribute %q=%q", attr.Key, attr.Value)
			}
		}
	}
	return nil
}

// Swap defines the "swap" event type.
type Swap struct {
	TxID               []byte
	Chain              []byte
	FromAddr           []byte
	ToAddr             []byte
	CoinAsset          []byte
	CoinAmount         int64
	Memo               []byte
	Pool               []byte
	PriceTarget        int64
	TradeSlip          int64
	LiquidityFee       int64
	LiquidityFeeInRune int64
}

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
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
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
	TxID        []byte
	Chain       []byte
	FromAddr    []byte
	ToAddr      []byte
	CoinAsset   []byte
	CoinAmount  int64
	Memo        []byte
	Pool        []byte
	StakeUnits  int64
	BasisPoints int64
	Asymmetry   float64
}

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
			e.CoinAsset, e.CoinAmount, err = parseCoin(attr.Value)
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

func parseCoin(b []byte) (asset []byte, amount int64, err error) {
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return nil, 0, errNoSep
	}
	asset = b[i+1:]
	amount, err = strconv.ParseInt(string(b[:i]), 10, 64)
	return
}
