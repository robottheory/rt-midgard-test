// Package event provides the blockchain data in a structured way.
//
// All asset amounts are fixed to 8 decimals. The resolution is made
// explicit with an E8 in the respective names.
//
// Numeric values are 64 bits wide, instead of the conventional 256
// bits used by most blockchains.
//
//	9 223 372 036 854 775 807  64-bit signed integer maximum
//	               00 000 000  decimals for fractions
//	   50 000 000 0·· ··· ···  500 M Rune total
//	    2 100 000 0·· ··· ···  21 M BitCoin total
//	   20 000 000 0·· ··· ···  200 M Ether total
package event

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/tendermint/tendermint/libs/kv"
)

// Asset Labels
const (
	// Native asset on THORChain.
	Rune = "THOR.RUNE"
	// Asset on Binance test net.
	rune67C = "BNB.RUNE-67C"
	// Asset on Binance main net.
	runeB1A = "BNB.RUNE-B1A"
)

// IsRune returns whether asset matches any of the supported $RUNE assets.
func IsRune(asset []byte) bool {
	switch string(asset) {
	case Rune, rune67C, runeB1A:
		return true
	}
	return false
}

// Rune Asset returns a matching RUNE asset given a running environment
// (Logic is copied from THORnode code)
func RuneAsset() string {
	if strings.EqualFold(os.Getenv("NATIVE"), "true") {
		return Rune
	}
	if strings.EqualFold(os.Getenv("NET"), "testnet") || strings.EqualFold(os.Getenv("NET"), "mocknet") {
		return rune67C
	}
	return runeB1A
}

// ParseAsset decomposes the notation.
//
//	asset  :≡ chain '.' symbol | symbol
//	symbol :≡ ticker '-' ID | ticker
//
func ParseAsset(asset []byte) (chain, ticker, id []byte) {
	i := bytes.IndexByte(asset, '.')
	if i > 0 {
		chain = asset[:i]
	}
	symbol := asset[i+1:]

	i = bytes.IndexByte(symbol, '-')
	if i < 0 {
		ticker = symbol
	} else {
		ticker = symbol[:i]
		id = symbol[i+1:]
	}
	return
}

// CoinSep is the separator for coin lists.
var coinSep = []byte{',', ' '}

type Amount struct {
	Asset []byte
	E8    int64
}

/*************************************************************/
/* Data models with Tendermint bindings in alphabetic order: */

// BUG(pascaldekloe): Duplicate keys in Tendermint transactions overwrite on another.

// ActiveVault defines the "ActiveVault" event type.
type ActiveVault struct {
	AddAsgardAddr []byte
}

// LoadTendermint adopts the attributes.
func (e *ActiveVault) LoadTendermint(attrs []kv.Pair) error {
	*e = ActiveVault{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "add new asgard vault":
			e.AddAsgardAddr = attr.Value

		default:
			log.Printf("unknown ActiveVault event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Add defines the "add" event type.
type Add struct {
	Tx       []byte
	Chain    []byte
	FromAddr []byte
	ToAddr   []byte
	Asset    []byte
	AssetE8  int64 // Asset quantity times 100 M
	Memo     []byte

	RuneE8 int64 // Number of runes times 100 M

	Pool []byte
}

// LoadTendermint adopts the attributes.
func (e *Add) LoadTendermint(attrs []kv.Pair) error {
	*e = Add{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.Tx = attr.Value
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
					e.RuneE8 = amountE8
				} else {
					e.AssetE8 = amountE8
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

// AsgardFundYggdrasil defines the "asgard_fund_yggdrasil" event type.
type AsgardFundYggdrasil struct {
	Tx       []byte // THORChain transaction identifier
	Asset    []byte
	AssetE8  int64  // Asset quantity times 100 M
	VaultKey []byte // public key of yggdrasil
}

// LoadTendermint adopts the attributes.
func (e *AsgardFundYggdrasil) LoadTendermint(attrs []kv.Pair) error {
	*e = AsgardFundYggdrasil{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {

		case "tx":
			e.Tx = attr.Value
		case "coins":
			e.Asset, e.AssetE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coins: %w", err)
			}
		case "pubkey":
			e.VaultKey = attr.Value

		default:
			log.Printf("unknown asgard_fund_yggdrasil event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Bond defines the "bond" event type.
type Bond struct {
	Tx       []byte
	Chain    []byte
	FromAddr []byte
	ToAddr   []byte
	Asset    []byte
	AssetE8  int64 // Asset quantity times 100 M
	Memo     []byte

	BoundType []byte
	E8        int64
}

// LoadTendermint adopts the attributes.
func (e *Bond) LoadTendermint(attrs []kv.Pair) error {
	*e = Bond{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.Tx = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AssetE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "amount":
			e.E8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed amount: %w", err)
			}
		case "bound_type":
			e.BoundType = attr.Value

		default:
			log.Printf("unknown bond event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Errata defines the "errata" event type.
type Errata struct {
	InTx    []byte
	Asset   []byte
	AssetE8 int64 // Asset quantity times 100 M
	RuneE8  int64 // Number of runes times 100 M
}

// LoadTendermint adopts the attributes.
func (e *Errata) LoadTendermint(attrs []kv.Pair) error {
	*e = Errata{}

	var flipAsset, flipRune bool

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "in_tx_id":
			e.InTx = attr.Value
		case "asset":
			e.Asset = attr.Value
		case "asset_amt":
			e.AssetE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed asset_amt: %w", err)
			}
		case "asset_add":
			add, err := strconv.ParseBool(string(attr.Value))
			if err != nil {
				return fmt.Errorf("malformed asset_add: %w", err)
			}
			flipAsset = !add
		case "rune_amt":
			e.RuneE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed rune_amt: %w", err)
			}
		case "rune_add":
			add, err := strconv.ParseBool(string(attr.Value))
			if err != nil {
				return fmt.Errorf("malformed rune_add: %w", err)
			}
			flipRune = !add
		default:
			log.Printf("unknown errata event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	if flipAsset {
		e.AssetE8 = -e.AssetE8
	}
	if flipRune {
		e.RuneE8 = -e.RuneE8
	}

	return nil
}

// Fee defines the "fee" event type, which records network fees applied to outbound transactions
type Fee struct {
	Tx         []byte // THORChain transaction identifier
	Asset      []byte
	AssetE8    int64 // Asset quantity times 100 M
	PoolDeduct int64 // rune quantity times 100 M
}

// LoadTendermint adopts the attributes.
func (e *Fee) LoadTendermint(attrs []kv.Pair) error {
	*e = Fee{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "tx_id":
			e.Tx = attr.Value
		case "coins":
			e.Asset, e.AssetE8, err = parseCoin(attr.Value)
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
	Asset   []byte
	AssetE8 int64 // Asset quantity times 100 M
	RuneE8  int64 // Number of runes times 100 M
	TxCount int64
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
			e.AssetE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed asset_amt: %w", err)
			}
		case "rune_amt":
			e.RuneE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
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

// InactiveVault defines the "InactiveVault" event type.
type InactiveVault struct {
	AddAsgardAddr []byte
}

// LoadTendermint adopts the attributes.
func (e *InactiveVault) LoadTendermint(attrs []kv.Pair) error {
	*e = InactiveVault{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "set asgard vault to inactive":
			e.AddAsgardAddr = attr.Value

		default:
			log.Printf("unknown InactiveVault event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Message defines the "message" event type.
type Message struct {
	FromAddr []byte // optional sender
	Action   []byte
}

// LoadTendermint adopts the attributes.
func (e *Message) LoadTendermint(attrs []kv.Pair) error {
	*e = Message{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "sender":
			e.FromAddr = attr.Value
		case "action":
			e.Action = attr.Value

		default:
			log.Printf("unknown message event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// NewNode defines the "new_node" event type.
type NewNode struct {
	NodeAddr []byte // THOR address
}

// LoadTendermint adopts the attributes.
func (e *NewNode) LoadTendermint(attrs []kv.Pair) error {
	*e = NewNode{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "address":
			e.NodeAddr = attr.Value
		default:
			log.Printf("unknown new_node event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Outbound defines the "outbound" event type, which records a transfer
// confirmation from pools. Each Swap, Unstake, UnBond or Refunds event is
// completed with an Outbound.
//
// All zeros on Tx are ignored, thus keeping a nil value. E.g., the Outbound of
// the “to RUNE swap” on double-swaps has no transaction ID.
type Outbound struct {
	Tx       []byte // THORChain transaction ID
	Chain    []byte // transfer backend ID
	FromAddr []byte // transfer pool address
	ToAddr   []byte // transfer contender address
	Asset    []byte // transfer unit ID
	AssetE8  int64  // transfer quantity times 100 M
	Memo     []byte // transfer description
	InTx     []byte // THORChain transaction ID reference
}

// LoadTendermint adopts the attributes.
func (e *Outbound) LoadTendermint(attrs []kv.Pair) error {
	*e = Outbound{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			// omit all-zero placeholders
			for _, c := range attr.Value {
				if c != '0' {
					e.Tx = attr.Value
					break
				}
			}
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AssetE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "in_tx_id":
			e.InTx = attr.Value

		default:
			log.Printf("unknown outbound event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Pool defines the "pool" event type.
type Pool struct {
	Asset  []byte
	Status []byte
}

// LoadTendermint adopts the attributes.
func (e *Pool) LoadTendermint(attrs []kv.Pair) error {
	*e = Pool{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "pool":
			e.Asset = attr.Value
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
	Tx         []byte
	Chain      []byte
	FromAddr   []byte
	ToAddr     []byte
	Asset      []byte
	AssetE8    int64 // Asset quantity times 100 M
	Asset2nd   []byte
	Asset2ndE8 int64 // Asset2 quantity times 100 M
	Memo       []byte

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
			e.Tx = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			v := attr.Value
			if i := bytes.Index(v, []byte{',', ' '}); i >= 0 {
				e.Asset2nd, e.Asset2ndE8, err = parseCoin(v[i+2:])
				if err != nil {
					return fmt.Errorf("malformed coin: %w", err)
				}

				v = v[:i]
			}
			e.Asset, e.AssetE8, err = parseCoin(v)
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
	Tx       []byte
	Chain    []byte // redundant to asset
	FromAddr []byte
	ToAddr   []byte // may have multiple, separated by space
	Asset    []byte
	AssetE8  int64 // Asset quantity times 100 M
	Memo     []byte

	Addr []byte
	E8   int64 // Number of runes times 100 M
}

// LoadTendermint adopts the attributes.
func (e *Reserve) LoadTendermint(attrs []kv.Pair) error {
	*e = Reserve{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		// thornode: common.Tx
		case "id":
			e.Tx = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AssetE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "contributor_address":
			e.Addr = attr.Value
		case "amount":
			e.E8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed amount: %w", err)
			}

		default:
			log.Printf("unknown reserve event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Rewards defines the "rewards" event type.
type Rewards struct {
	BondE8 int64 // rune amount times 100 M
	// PerPool has the RUNE amounts specified per pool (in .Asset).
	PerPool []Amount
}

// LoadTendermint adopts the attributes.
func (e *Rewards) LoadTendermint(attrs []kv.Pair) error {
	*e = Rewards{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "bond_reward":
			e.BondE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed bond_reward: %w", err)
			}

		default:
			v, err := strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				log.Printf("unknown rewards event attribute %q=%q", attr.Key, attr.Value)
				break
			}
			e.PerPool = append(e.PerPool, Amount{attr.Key, v})
		}
	}

	return nil
}

// SetIPAddr defines the "set_ip_address" event type.
type SetIPAddress struct {
	NodeAddr []byte // THOR address
	IPAddr   []byte
}

// LoadTendermint adopts the attributes.
func (e *SetIPAddress) LoadTendermint(attrs []kv.Pair) error {
	*e = SetIPAddress{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "thor_address":
			e.NodeAddr = attr.Value
		case "address":
			e.IPAddr = attr.Value
		default:
			log.Printf("unknown set_ip_address event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// SetMimir defines the "set_mimir" event type.
type SetMimir struct {
	Key   []byte
	Value []byte
}

// LoadTendermint adopts the attributes.
func (e *SetMimir) LoadTendermint(attrs []kv.Pair) error {
	*e = SetMimir{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "key":
			e.Key = attr.Value
		case "value":
			e.Value = attr.Value

		default:
			log.Printf("unknown set_mimir event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// SetNodeKeys defines the "set_node_keys" event type.
type SetNodeKeys struct {
	NodeAddr           []byte // THOR address
	Secp256k1          []byte // public key
	Ed25519            []byte // public key
	ValidatorConsensus []byte // public key
}

// LoadTendermint adopts the attributes.
func (e *SetNodeKeys) LoadTendermint(attrs []kv.Pair) error {
	*e = SetNodeKeys{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "node_address":
			e.NodeAddr = attr.Value
		case "node_secp256k1_pubkey":
			e.Secp256k1 = attr.Value
		case "node_ed25519_pubkey":
			e.Ed25519 = attr.Value
		case "validator_consensus_pub_key":
			e.ValidatorConsensus = attr.Value
		default:
			log.Printf("unknown set_node_keys event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// SetVersion defines the "set_version" event type.
type SetVersion struct {
	NodeAddr []byte // THOR address
	Version  string
}

// LoadTendermint adopts the attributes.
func (e *SetVersion) LoadTendermint(attrs []kv.Pair) error {
	*e = SetVersion{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "thor_address":
			e.NodeAddr = attr.Value
		case "version":
			e.Version = string(attr.Value)
		default:
			log.Printf("unknown set_version event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Stake defines the "stake" event type, which records a participation result."
type Stake struct {
	Pool       []byte // asset ID
	AssetTx    []byte // transfer transaction ID (may equal RuneTx)
	AssetChain []byte // transfer backend ID
	AssetAddr  []byte // pool contender address
	AssetE8    int64  // transfer asset quantity times 100 M
	RuneTx     []byte // pool transaction ID
	RuneChain  []byte // pool backend ID
	RuneAddr   []byte // pool contender address
	RuneE8     int64  // pool transaction quantity times 100 M
	StakeUnits int64  // pool's liquidiy tokens—gained quantity
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
		case "THORChain_txid", "BNBChain_txid", "BNB_txid": // BNBChain for Binance test & main net
			e.RuneTx = attr.Value
			e.RuneChain = attr.Key[:len(attr.Key)-len(txIDSuffix)]
		case "rune_address":
			e.RuneAddr = attr.Value
		case "rune_amount":
			e.RuneE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed rune_amount: %w", err)
			}
		case "asset_amount":
			e.AssetE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed asset_amount: %w", err)
			}
		case "asset_address":
			e.AssetAddr = attr.Value

		default:
			switch {
			case bytes.HasSuffix(attr.Key, txIDSuffix):
				if e.AssetChain != nil {
					// It should not be that there are two *_txid attrs of which neither is the RUNE one
					return fmt.Errorf("%q preceded by %q%s", attr.Key, e.AssetChain, txIDSuffix)
				}
				e.AssetChain = attr.Key[:len(attr.Key)-len(txIDSuffix)]

				e.AssetTx = attr.Value

			default:
				log.Printf("unknown stake event attribute %q=%q", attr.Key, attr.Value)
			}
		}
	}

	if e.AssetTx == nil {
		// if the asset and RUNE were sent in the same tx, then AssetTx will end up empty
		e.AssetTx = e.RuneTx
		e.AssetChain = e.RuneChain
	}

	return nil
}

// Slash defines the "slash" event type.
type Slash struct {
	Pool    []byte
	Amounts []Amount
}

// LoadTendermint adopts the attributes.
func (e *Slash) LoadTendermint(attrs []kv.Pair) error {
	*e = Slash{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "pool":
			e.Pool = attr.Value

		default:
			v, err := strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				log.Printf("unknown slash event attribute %q=%q", attr.Key, attr.Value)
				break
			}
			e.Amounts = append(e.Amounts, Amount{attr.Key, v})
		}
	}

	return nil
}

// Swap defines the "swap" event type, which records an exchange
// between the .Pool asset and RUNE.
//
// FromAsset is the input unit of a Swap. The value equals .Pool
// if and only if the trader sells the pool's asset for RUNE. In
// all other cases .FromAsset will be a RUNE, because the trader
// buys .Pool asset.
//
// The liquidity fee is included. Network fees are recorded as
// separate Fee events (with a matching .Tx value).
type Swap struct {
	Tx             []byte // THOR transaction identifier
	Chain          []byte // backend identifier
	FromAddr       []byte // input address on Chain
	ToAddr         []byte // output address on Chain
	FromAsset      []byte // input unit
	FromE8         int64  // FromAsset quantity times 100 M
	ToE8           int64  // ToAsset quantity times 100 M
	Memo           []byte // encoded parameters
	Pool           []byte // asset identifier
	ToE8Min        int64  // output quantity constraint
	TradeSlipBP    int64  // ‱ the trader experienced
	LiqFeeE8       int64  // Pool asset quantity times 100 M
	LiqFeeInRuneE8 int64  // equivalent in RUNE times 100 M
}

// LoadTendermint adopts the attributes.
func (e *Swap) LoadTendermint(attrs []kv.Pair) error {
	*e = Swap{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.Tx = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.FromAsset, e.FromE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coins: %w", err)
			}
		case "emit_asset":
			_, e.ToE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed emit_asset: %w", err)
			}
		case "memo":
			e.Memo = attr.Value

		case "pool":
			e.Pool = attr.Value
		case "price_target":
			e.ToE8Min, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed price_target: %w", err)
			}
		case "trade_slip":
			e.TradeSlipBP, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed trade_slip: %w", err)
			}
		case "liquidity_fee":
			e.LiqFeeE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed liquidity_fee: %w", err)
			}
		case "liquidity_fee_in_rune":
			e.LiqFeeInRuneE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed liquidity_fee_in_rune: %w", err)
			}

		default:
			log.Printf("unknown swap event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// ToRune returns whether the swap output is RUNE.
func (e *Swap) ToRune() bool { return bytes.Equal(e.Pool, e.FromAsset) }

// FromRune returns whether the swap input is RUNE.
func (e *Swap) FromRune() bool { return !e.ToRune() }

// DoubleAsset returns the follow-up pool or nil. Follow-ups occur in so-called
// double-swaps, whereby the trader sells .Pool asset with this event, and then
// consecutively buys DoubleAsset in another event (with the same .Tx).
func (e *Swap) DoubleAsset() (asset []byte) {
	if e.ToRune() {
		params := bytes.SplitN(e.Memo, []byte{':'}, 3)
		if len(params) > 1 && !bytes.Equal(params[1], e.Pool) {
			return params[1]
		}
	}
	return nil
}

// Transfer defines the "transfer" event type.
type Transfer struct {
	FromAddr []byte // sender
	ToAddr   []byte // recipient
	RuneE8   int64  // Amount of RUNE times 100 M
}

// LoadTendermint adopts the attributes.
func (e *Transfer) LoadTendermint(attrs []kv.Pair) error {
	*e = Transfer{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "sender":
			e.FromAddr = attr.Value
		case "recipient":
			e.ToAddr = attr.Key
		case "amount":
			if !bytes.HasSuffix(attr.Value, []byte("thor")) {
				return fmt.Errorf("unknown amount unit %q", attr.Value)
			}
			e.RuneE8, err = strconv.ParseInt(string(attr.Value[:len(attr.Value)-4]), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed amount: %w", err)
			}

		default:
			log.Printf("unknown transfer event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// Unstake defines the "unstake" event type, which records a pool withdrawal request.
// Requests are made by wiring a (probably small) “donation” to the reserve.
// The actual withdrawal that follows is confirmed by an Outbound.
type Unstake struct {
	Tx          []byte  // THORChain transaction ID
	Chain       []byte  // transfer backend ID
	FromAddr    []byte  // transfer staker address
	ToAddr      []byte  // transfer pool address
	Asset       []byte  // transfer unit ID
	AssetE8     int64   // transfer quantity times 100 M
	EmitAssetE8 int64   // asset amount withdrawn
	EmitRuneE8  int64   // rune amount withdrawn
	Memo        []byte  // description code which triggered the event
	Pool        []byte  // asset ID
	StakeUnits  int64   // pool's liquidiy tokens—lost quantity
	BasisPoints int64   // ‱ of what?
	Asymmetry   float64 // lossy conversion of what?
}

// LoadTendermint adopts the attributes.
func (e *Unstake) LoadTendermint(attrs []kv.Pair) error {
	*e = Unstake{}

	for _, attr := range attrs {
		var err error
		switch string(attr.Key) {
		case "id":
			e.Tx = attr.Value
		case "chain":
			e.Chain = attr.Value
		case "from":
			e.FromAddr = attr.Value
		case "to":
			e.ToAddr = attr.Value
		case "coin":
			e.Asset, e.AssetE8, err = parseCoin(attr.Value)
			if err != nil {
				return fmt.Errorf("malformed coin: %w", err)
			}
		case "emit_asset":
			e.EmitAssetE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed emit_asset: %w", err)
			}
		case "emit_rune":
			e.EmitRuneE8, err = strconv.ParseInt(string(attr.Value), 10, 64)
			if err != nil {
				return fmt.Errorf("malformed emit_asset: %w", err)
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

// UpdateNodeAccountStatus defines the "UpdateNodeAccountStatus" event type.
type UpdateNodeAccountStatus struct {
	NodeAddr []byte // THORChain address
	Former   []byte // previous status label
	Current  []byte // new status label
}

// LoadTendermint adopts the attributes.
func (e *UpdateNodeAccountStatus) LoadTendermint(attrs []kv.Pair) error {
	*e = UpdateNodeAccountStatus{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "Address":
			e.NodeAddr = attr.Value
		case "Former:":
			e.Former = attr.Value
		case "Current:":
			e.Current = attr.Value

		default:
			log.Printf("unknown UpdateNodeAccountStatus event attribute %q=%q", attr.Key, attr.Value)
		}
	}

	return nil
}

// ValidatorRequestLeave defines the "validator_request_leave" event type.
type ValidatorRequestLeave struct {
	Tx       []byte // THORChain transaction identifier
	FromAddr []byte // signer THOR node
	NodeAddr []byte // subject THOR node
}

// LoadTendermint adopts the attributes.
func (e *ValidatorRequestLeave) LoadTendermint(attrs []kv.Pair) error {
	*e = ValidatorRequestLeave{}

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "tx":
			e.Tx = attr.Value
		case "signer bnb address":
			e.FromAddr = attr.Value
		case "destination":
			e.NodeAddr = attr.Value

		default:
			log.Printf("unknown validator_request_leave event attribute %q=%q", attr.Key, attr.Value)
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
