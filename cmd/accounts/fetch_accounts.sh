#!/bin/bash
set -euo pipefail
fetch()
{
	server=$1
	endpoint=$2
	offset=$3

	curl --insecure $server/$endpoint?pagination.offset=$offset
}
itemcount()
{
	input=$1
	
 	jq ".accounts[].address" $input | wc -l
}

server=$1
endpoint=cosmos/auth/v1beta1/accounts
offset=$2
count=-1
while [[ $count != 0 ]]
do
	output=result_$offset
	fetch $server $endpoint $offset > $output
	count=$( itemcount $output )
	offset=$[offset+count]
	sleep 10
done
