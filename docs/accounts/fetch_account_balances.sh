#!/bin/bash
set -euo pipefail
address=$1
server=${2:-https://thornode.ninerealms.com}
endpoint="cosmos/bank/v1beta1/balances"
height="x-cosmos-block-height: $3"
curl --insecure ${server}/${endpoint}/${address} -H "${height}"
