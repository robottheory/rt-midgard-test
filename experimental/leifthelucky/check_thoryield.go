// This tool shows various views and queries used by the aggregate mechanisms.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

const usageString = `Check ThorYield calculations for a given address.
Usage:
$ go run ./check_thoryield [flags]
`

var midgard = flag.String("midgard", "https://midgard.thorchain.info", "Midgard url")
var addr = flag.String("address", "thor1wfe7hsuvup27lx04p5al4zlcnx6elsnyft7dzm",
	"Address to check: either RUNE address or an asset address")
var pool = flag.String("pool", "", "Liquidity pool to check.")

func init() {
	flag.Usage = func() {
		fmt.Println(usageString)
		flag.PrintDefaults()
	}
}

var THOR_YIELD = "https://api.thoryield.net/lp/redacted/%s?pool=%s"

var MEMBER = "/v2/member/%s"
var LIQUIDITY_ACTIONS = "/v2/actions?address=%s,type=%s,limit=100000"
var DEPTH_HISTORY = "/v2/history/depths/%s?to=%d"

func fetchUrl(url string) (*[]byte, error) {
	return fetchUrlWithHeaders(url, make(map[string][]string))
}

func fetchUrlWithHeaders(url_ string, headers map[string][]string) (*[]byte, error) {
	client := &http.Client{}
	urll, _ := url.Parse(url_)
	req := http.Request{
		Method: "GET",
		Header: headers,
		URL:    urll,
	}
	resp, err := client.Do(&req)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Failed to fetch: %s",
			url_)+"\nError: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read resp.Body: %w", err)
		}
		return &bodyBytes, nil
	}
	return nil, errors.New(fmt.Sprintf("HTTP Fetch not OK: ", resp.StatusCode))
}

func fetchMemberDetails(addr string) (*oapigen.MemberDetails, error) {
	url := *midgard + fmt.Sprintf(MEMBER, addr)
	bytes, err := fetchUrl(url)
	if err != nil {
		return nil, err
	}
	var r oapigen.MemberDetails
	err = json.Unmarshal(*bytes, &r)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Failed to parse response: %s\n",
			string(*bytes))+"Error: %w", err)
	}
	return &r, nil
}

func fetchLiquidityActions(addr string, filter string) (*oapigen.ActionsResponse, error) {
	url := *midgard + fmt.Sprintf(LIQUIDITY_ACTIONS, addr, filter)
	bytes, err := fetchUrl(url)
	if err != nil {
		return nil, err
	}
	var r oapigen.ActionsResponse
	err = json.Unmarshal(*bytes, &r)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Failed to parse response: %s\n",
			string(*bytes))+"Error: %w", err)
	}
	return &r, err
}

func fetchDepthHistory(pool string, to int64) (*oapigen.DepthHistory, error) {
	url := *midgard + fmt.Sprintf(DEPTH_HISTORY, pool, to)
	bytes, err := fetchUrl(url)
	if err != nil {
		return nil, err
	}
	var r oapigen.DepthHistory
	err = json.Unmarshal(*bytes, &r)
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Failed to parse response: %s\n",
			string(*bytes))+"Error: %w", err)
	}
	return &r, err
}

type LPGains struct {
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time`
}

type ThorYieldData struct {
	HistoricLPGains       []LPGains `json:"historic_lp_gains"`
	LPShare               float64   `json:"lp_share"`
	LPUnits               float64   `json:"lp_units"`
	LPUnitsPool           float64   `json:"lp_units_pool"`
	RuneAmountAdded       float64   `json:"rune_amount_added"`
	AssetAmountAdded      float64   `json:"asset_amount_added"`
	RuneAmountRedeemable  float64   `json:"rune_amount_redeemable"`
	AssetAmountRedeemable float64   `json:"asset_amount_redeemable"`
	RuneDepth             float64   `json:"rune_depth"`
	AssetDepth            float64   `json:"asset_depth"`
}

func fetchThorYieldResponse(addr string, pool string) (*ThorYieldData, error) {
	key := os.Getenv("THORYIELD_API_KEY")
	if key == "" {
		return nil, errors.New(
			"ThorYield API Key required. Set ENV variable: THORYIELD_API_KEY")
	}
	url := fmt.Sprintf(THOR_YIELD, addr, pool)
	fmt.Printf("Fetching: %s\n", url)
	bytes, err := fetchUrlWithHeaders(url,
		map[string][]string{"key": {key}})
	if err != nil {
		return nil, err
	}
	// The response json looks like this:
	//   {"BTC.BTC": {...},
	//    "ETH.ETH": {...}}
	//
	// Since we don't want to define our struct with all possible pools,
	// we first parse the message into a map[string]interface{} then
	// we pull out just the data for the relevant pool and unmarshal that.
	var kv map[string]interface{}
	err = json.Unmarshal(*bytes, &kv)
	if err != nil {
		return nil, err
	}
	p, ok := kv[pool]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", pool)
	}
	ty := ThorYieldData{}
	// TODO(leifthelucky): Surely there's a better way than marshaling the
	// data back into json just so it can be unmarshaled again.
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &ty)
	if err != nil {
		return nil, err
	}
	return &ty, nil
}

func si64tof64(s string) float64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Fatal("Failed to parse: ", err)
	}
	return float64(i)
}

func main() {
	// lp_return_usd =
	// + redeemable_rune * latest_rune_price_usd + redeemable_asset * latest_asset_price_usd   # /v2/member; /v2/history/depths/{pool}
	// + sum_{b : block with withdraw rune event} withdraw_rune_b * rune_price_usd_b   # /actions?address=<thor..>&type=addLiquidity; /v2/history/depths/{pool}
	// + sum_{b : block with withdraw asset event} withdraw_asset_b * asset_price_usd_b    # /actions?address=<thor..>&type=withdraw; /v2/history/depths/{pool}
	// - sum_{b : block with add rune event} added_rune_b * asset_rune_usd_b   # /actions?address=<thor..>&type=addLiquidity; /v2/history/depths/{pool}
	// - sum_{b : block with add asset event} added_asset_b * asset_price_usd_b    # /actions?address=<thor..>&type=addLiquidity; /v2/history/depths/{pool}
	flag.Parse()

	m, err := fetchMemberDetails(*addr)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%+v\n", m)
	//a, err := fetchLiquidityActions(*addr, "addLiquidity,withdraw")
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%+v\n", a)
	fmt.Printf("Stats for: %s\n", *addr)
	for _, memberPool := range m.Pools {
		if *pool != "" && *pool != memberPool.Pool {
			continue
		}
		fmt.Printf("Pool: %s\n", memberPool.Pool)

		// Print member pool info.
		b, err := json.MarshalIndent(memberPool, "", "  ")
		if err != nil {
			log.Fatal("Failed to marshal json: ", err)
		}
		fmt.Println(string(b))

		ty, err := fetchThorYieldResponse(*addr, memberPool.Pool)
		if err != nil {
			log.Fatal("Failed to fetch ThorYield response: ", err)
		}

		toTimestamp := ty.HistoricLPGains[len(ty.HistoricLPGains)-1].EndTime / 1000
		fmt.Printf("Fetching pool depth history until: %d\n", toTimestamp)
		p, err := fetchDepthHistory(memberPool.Pool, toTimestamp)
		if err != nil {
			log.Fatal("Failed to fetch pool: ", memberPool.Pool, " Error:", err)
		}
		b, err = json.MarshalIndent(p, "", "  ")
		if err != nil {
			log.Fatal("Failed to marshal json: ", err)
		}
		fmt.Println(string(b))

		// Print Redeemable values.
		stake := si64tof64(memberPool.LiquidityUnits)
		interval := p.Intervals[len(p.Intervals)-1]
		totalStake := si64tof64(interval.Units)
		share := float64(stake) / float64(totalStake)

		fmt.Printf("Added:"+
			"\n  Asset: %f\n  Rune: %f\n\n",
			si64tof64(memberPool.AssetAdded)/1e8,
			si64tof64(memberPool.RuneAdded)/1e8)
		fmt.Printf("Added (ThorYield):"+
			"\n  Asset: %f\n  Rune: %f\n\n",
			ty.AssetAmountAdded,
			ty.RuneAmountAdded)
		fmt.Printf("Liquidity units:\n  %f\n\n", si64tof64(memberPool.LiquidityUnits)/1e8)
		fmt.Printf("Liquidity units (ThorYield):\n  %f\n\n", ty.LPUnits)
		fmt.Printf("Redeemable:"+
			"\n  Asset: %f (%f%% of %f)\n  Rune: %f (%f%% of %f)\n\n",
			share*si64tof64(interval.AssetDepth)/1e8, share*100, si64tof64(interval.AssetDepth)/1e8,
			share*si64tof64(interval.RuneDepth)/1e8, share*100, si64tof64(interval.RuneDepth)/1e8)
		fmt.Printf("Redeemable (ThorYield):"+
			"\n  Asset: %f (%f%% of %f)\n  Rune: %f (%f%% of %f)\n\n",
			ty.AssetAmountRedeemable, ty.LPShare*100, ty.AssetDepth,
			ty.RuneAmountRedeemable, ty.LPShare*100, ty.RuneDepth)
	}
}
