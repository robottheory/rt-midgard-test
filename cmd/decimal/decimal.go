package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
	"gopkg.in/yaml.v3"
)

// If you want to update decimal of the pools, run this script in the command line: `go run ./cmd/decimal`
// If the script succeeds it will create the result in the `resources/decimals/decimals.yaml`

type ResultMap map[string]SingleResult

type SingleResult struct {
	NativeDecimals int64    `json:"decimals"` // -1 means that only the asset name was observed without the decimal count.
	AssetSeen      []string `json:"asset_seen"`
	DecimalSource  []string `json:"decimal_source"`
}

func main() {
	midlog.LogCommandLine()
	config.ReadGlobal()

	thorNodePools := readFromThorNodePools()
	midgardPools := readFromMidgardPools()
	manualPools := readManualJson()

	finalMergedPools := make(ResultMap)
	finalMergedPools.mergeFrom(thorNodePools, midgardPools, manualPools)
	finalMergedPools.mergeFrom(getERC20decimal(finalMergedPools))

	content, err := yaml.Marshal(finalMergedPools)
	if err != nil {
		midlog.FatalE(err, "Can't Marshal the resulted decimal pools to yaml.")
	}

	err = ioutil.WriteFile("./resources/decimals/decimals.yaml", content, 0644)
	if err != nil {
		midlog.FatalE(err, "Can't Marshal pools to decimals yaml.")
	}
}

type PoolsResponse struct {
	Pools []struct {
		Asset   string `json:"asset"`
		Decimal int64  `json:"decimals"` // This field is might be filled only in the ThorNode response
	}
}

func readFromThorNodePools() ResultMap {
	urls := map[string]string{
		"thornode-mainnet":  "https://thornode.ninerealms.com",
		"thornode-stagenet": "https://stagenet-thornode.ninerealms.com",
		"thornode-testnet":  "https://testnet.thornode.thorchain.info",
	}

	pools := ResultMap{}
	for net, url := range urls {
		var res PoolsResponse
		queryEndpoint(url, "/thorchain/pools", &res.Pools)
		pools.mergeFrom(res.toResultMap(net))
	}

	return pools
}

func readFromMidgardPools() ResultMap {
	urls := map[string]string{
		"midgard-mainnet":  "https://midgard.thorchain.info",
		"midgard-stagenet": "https://stagenet-midgard.ninerealms.com",
		"midgard-testnet":  "https://testnet.midgard.thorchain.info/",
	}

	pools := ResultMap{}
	for net, url := range urls {
		var res PoolsResponse
		queryEndpoint(url, "/v2/pools", &res.Pools)
		pools.mergeFrom(res.toResultMap(net))
	}

	return pools
}

func (pr PoolsResponse) toResultMap(network string) ResultMap {
	mapPools := ResultMap{}
	for _, p := range pr.Pools {
		decimals := p.Decimal
		decimalSource := []string{}
		if decimals == 0 {
			decimals = -1
		} else if 0 < decimals {
			decimalSource = append(decimalSource, network)
		}
		mapPools[p.Asset] = SingleResult{
			NativeDecimals: decimals,
			AssetSeen:      []string{network},
			DecimalSource:  decimalSource,
		}
	}
	return mapPools
}

func (to *ResultMap) mergeFrom(from ...ResultMap) {
	for _, f := range from {
		for poolName, fromInfo := range f {
			toInfo, ok := (*to)[poolName]
			if !ok {
				toInfo.NativeDecimals = -1
			}
			toInfo.AssetSeen = append(toInfo.AssetSeen, fromInfo.AssetSeen...)
			toInfo.DecimalSource = append(toInfo.DecimalSource, fromInfo.DecimalSource...)
			if toInfo.DecimalSource == nil {
				toInfo.DecimalSource = []string{}
			}
			if toInfo.NativeDecimals == -1 {
				toInfo.NativeDecimals = fromInfo.NativeDecimals
			} else {
				if -1 < fromInfo.NativeDecimals && fromInfo.NativeDecimals != toInfo.NativeDecimals {
					midlog.Fatal(fmt.Sprintf(
						"The %s source has %d decimal which is different than %d decimals on %v",
						fromInfo.AssetSeen,
						fromInfo.NativeDecimals,
						toInfo.NativeDecimals,
						toInfo.AssetSeen))
				}
			}
			(*to)[poolName] = toInfo
		}
	}
}

func queryEndpoint(urlAddress string, urlPath string, dest interface{}) {
	url := urlAddress + urlPath
	midlog.DebugF("Querying the endpoint: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		midlog.FatalE(err, fmt.Sprintf("Error while querying endpoint: %s", url+urlPath))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		midlog.Fatal("Cannot read the body of the response")
	}

	err = json.Unmarshal(body, dest)
	if err != nil {
		midlog.FatalE(err, fmt.Sprintf("Error while querying endpoint: %s", url+urlPath))
	}

}

func queryEthplorerAsset(assetAddress string) int64 {
	url := fmt.Sprintf("https://api.ethplorer.io/getTokenInfo/%s?apiKey=freekey", assetAddress)

	midlog.DebugF("Querying Ethplorer: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		midlog.FatalE(err, "Error querying Ethplorer")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		midlog.Fatal("Can't read the reponse body.")
	}

	var dest EthResponse
	err = json.Unmarshal(body, &dest)
	if err != nil {
		midlog.WarnF("Json unmarshal error for url: %s", url)
		midlog.FatalE(err, "Error unmarshalling ThorNode response")
	}

	decimal, err := strconv.ParseInt(dest.Decimals, 10, 64)
	if err != nil {
		midlog.FatalE(err, "Can't parse the decimal")
	}

	return decimal
}

func queryRopstenDecimalAsset(assetAddress string) int64 {
	url := "https://ethereum-ropsten-rpc.allthatnode.com"

	midlog.DebugF("Querying Ropsten json-rpc: %s for %s", url, assetAddress)

	payload := strings.NewReader(fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "eth_call",
		"params": [
			{
				"data": "0x313ce567",
				"to": "%s"
			},
			"latest"
		]
	}`, assetAddress))

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		midlog.FatalE(err, "Error on requesting to json-rpc")
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		midlog.FatalE(err, "Error querying json-rpc node")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	var dest EthResponse
	err = json.Unmarshal(body, &dest)
	if err != nil {
		midlog.WarnF("Json unmarshal error for url: %s", url)
		midlog.FatalE(err, "Error unmarshalling ThorNode response")
	}

	return hexToInt(dest.Result)
}

type EthResponse struct {
	Decimals string `json:"decimals"`
	Result   string `json:"result"`
}

func isTestnet(networks []string) bool {
	for _, v := range networks {
		if strings.Contains(v, "testnet") {
			return true
		}
	}
	return false
}

func hexToInt(hexaString string) int64 {
	// replace 0x or 0X with empty String
	numberStr := strings.Replace(hexaString, "0x", "", -1)
	numberStr = strings.Replace(numberStr, "0X", "", -1)

	number, err := strconv.ParseInt(numberStr, 16, 64)
	if err != nil {
		midlog.FatalE(err, "Can't parse hexadecimal to int64")
	}

	return number
}

func getERC20decimal(pools ResultMap) ResultMap {
	ercMap := make(map[string]SingleResult)
	cnt := 0
	for k, p := range pools {
		if strings.HasPrefix(k, "ETH") && k != "ETH.ETH" {
			r := strings.Split(k, "-")
			var decimal int64
			if isTestnet(p.AssetSeen) {
				decimal = queryRopstenDecimalAsset(r[1])
			} else {
				decimal = queryEthplorerAsset(r[1])
			}
			if decimal != 0 && decimal != -1 {
				ercMap[k] = SingleResult{
					NativeDecimals: decimal,
					AssetSeen:      []string{},
					DecimalSource:  []string{"ERC20"},
				}
			}
			cnt++
			// sleeps for 1 seconds to aviod Freekey limit
			if cnt%2 == 0 {
				time.Sleep(1 * time.Second)
			}
		}
	}

	return ercMap
}

func readManualJson() ResultMap {
	yamlFile, err := os.Open("./cmd/decimal/manual.yaml")
	manualResult := make(ResultMap)
	if err != nil {
		midlog.Fatal("There was no manual.yaml file")
		return manualResult
	}
	defer yamlFile.Close()

	var rawPools map[string]int64
	if err == nil {
		rawData, err := ioutil.ReadAll(yamlFile)
		if err != nil {
			midlog.FatalE(err, "Can't read manual.yaml")
		}
		err = yaml.Unmarshal(rawData, &rawPools)
		if err != nil {
			midlog.FatalE(err, "Can't Unmarshal manual pools yaml.")
		}
	}

	for p, v := range rawPools {
		manualResult[p] = SingleResult{
			NativeDecimals: v,
			AssetSeen:      []string{"constants"},
			DecimalSource:  []string{"constants"},
		}
	}

	return manualResult
}
