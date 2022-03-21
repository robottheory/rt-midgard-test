'use strict';
// main namespace g = "global"
var g = { m: {}, v: {}, c: {} };

g.c.queryEarnings = function () {
    const form = document.forms["lp"];

    const midgardUrl = "https://midgard.ninerealms.com/v2";
    const poolInput = form.pool.value;
    const addressInput = form.address.value;

    // TODO(leifthelucky): For consistency, the pool and member details data
    // should be requested as of a certain block height or timestamp. This
    // ensures not only correct numbers, but reproducibility, critical for
    // debugging. Unfortunately, the Midgard /member and /pool endpoints do
    // not accept a "block" parameter.
    let memberDetailsUrl = `${midgardUrl}/member/${addressInput}`;
    let poolUrl = `${midgardUrl}/pool/${poolInput}`;
    Promise.all([
        fetch(new Request(memberDetailsUrl)),
        fetch(new Request(poolUrl)),
    ]).then(function (responses) {
        return Promise.all(responses.map(function (res) {
            if (!res.ok) {
                throw new Error(`HTTP error! status: ${res.status}`);
            }
            return res.json();
        }));
    }).then(function (data) {
        if (data.length !== 2) {
            throw new Error(`Wrong number of responses.`);
        }
        const memberDetailsData = data[0]
        const poolData = data[1]
        for (const memberPoolData of memberDetailsData.pools) {
            if (memberPoolData.pool == poolInput) {
                const addrRune = addressInput.startsWith('thor') ? addressInput : memberPoolData.runeAddress;
                const addrAsset = !addressInput.startsWith('thor') ? addressInput : memberPoolData.assetAddress;
                // TODO(leifthelucky): Implement a full fetch of actions (they are paginated)
                // with max 50 per page.
                let actionsUrl = `${midgardUrl}/actions?address=${addrRune},${addrAsset}` +
                    `&type=addLiquidity,withdraw`;
                fetch(new Request(actionsUrl)).then(function (res) {
                    if (!res.ok) {
                        throw new Error(`HTTP error! status: ${res.status}`);
                    }
                    return res.json();
                }).then(function (actionsData) {
                    // Fetch price data for all blocks appearing in the actions data.
                    const historyBaseUrl = `${midgardUrl}/history/depths/${memberPoolData.pool}`
                    let historyPromises = []
                    actionsData.actions.forEach(function (a) {
                        if (!a.pools.find(p => p == poolInput)) {
                            return;
                        }
                        const dNano = a.date;
                        const dSec = Math.floor(dNano / 1e9);
                        const url = historyBaseUrl + `?from=${dSec}&to=${dSec}`;
                        historyPromises.push(fetch(new Request(url)));
                    });
                    Promise.all(historyPromises).then(function (responses) {
                        return Promise.all(responses.map(function (res) {
                            if (!res.ok) {
                                throw new Error(`HTTP error! status: ${res.status}`);
                            }
                            return res.json();
                        }));
                    }).then(function (historyData) {
                        let assetPriceInRuneByTime = {}
                        historyData.map(function (item) {
                            assetPriceInRuneByTime[item.meta.startTime] = item.intervals[0].assetPrice;
                        });
                        g.m.LPLiquidity.update(memberPoolData, poolData, actionsData, assetPriceInRuneByTime);
                        document.getElementById("view").innerHTML =
                            `<pre>${JSON.stringify(g.m.LPLiquidity, null, '\t')}</pre>`
                    });
                });
                break;
            }
        }
    }).catch(function (error) {
        console.log(error);
    });
}