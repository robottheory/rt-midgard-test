[![pipeline status](https://gitlab.com/thorchain/midgard/badges/master/pipeline.svg)](https://gitlab.com/thorchain/midgard/commits/master)


# Midgard API

Midgard is a layer 2 REST API that provides front-end consumers with semi real-time rolled up data and analytics of the THORChain network. Most requests to the network will come through Midgard. This daemon is here to keep the chain itself from fielding large quantities of requests. You can think of it as a “read-only slave” to the chain. This keeps the resources of the network focused on processing transactions.



### Run Midgard

The daemon needs PostgreSQL with the TimeScale extension.

```sh
docker-compose up --build -d pg
```

Now you can launch a local instance directly from the sources.

```sh
mkdir tmp # files in this directory are ignored by git
cp config/config.json tmp/config.json # make changes to config.json as necessary for your local machine
go run ./cmd/midgard tmp/config.json
```

`config/config.json` asumes you are running a ThorNode on localhost. If that is not the case or if you want to develop against a specific network, you may want to go to the [network's seed url](https://docs.thorchain.org/developers/connecting-to-thorchain) and pick a node ip from there to replace the host in `tendermint_url` and `thornode_url` with that ip. Though, if you expect that you'll want to sync up Midgard more than once, which is expected if you are planning to do any development, we kindly ask you to run a local ThorNode, see the next section.

Midgard populates the database with content from the blockchain.
Progress is traceable with the Prometheus Metrics propagated on
<http://localhost:8080/debug/metrics>, specifically the measurements
`midgard_chain_cursor_height` v.s. `midgard_chain_height`.

Open <http://localhost:8080/v2> in your browser for the GraphQL UI. ✨

### Running local ThorNode

To work on Midgard we don't need or want a proper validator setup, just the full thornode that follows and syncs the thorchain locally.

Clone the thornode repo from: https://gitlab.com/thorchain/thornode

Look up the current version and check it out. If you need the latest verion you are probably fine using the `chaosnet-multichain` branch.

Start the thornode by running `make run-fullnode` from `build/docker/mainnet`.

IMPORTANT! This will create a docker container named `thornode` and will store data in your home directory, under `~/.thornode`. If you have anything important in one or the other, backup first!

To summarize:

```sh
git clone https://gitlab.com/thorchain/thornode.git
cd thornode
git checkout chaosnet-multichain
cd build/docker/mainnet
make run-fullnode
```

For midgard config use:

```json
    "tendermint_url": "http://localhost:26657/websocket",
    "thornode_url": "http://localhost:1317/thorchain",
```

#### Upgrading local ThorNode

When the network switches to a newer version your local thornode will stop working:
the docker container will be in a crash loop. To upgrade, remove the container, the docker image,
pull, and restart:

```sh
docker stop thornode
docker rm thornode
docker rmi registry.gitlab.com/thorchain/thornode:mainnet
cd thornode/build/docker/mainnet
git pull
make run-fullnode
```

### Websockets

Websockets is an experimental feature supported for Linux only. If you need to use it for develop using a different OS you may need to run Midgard using Docker.

### Config

Configuration is loaded from a `.json` file. Default is in `config/config.json`.

Overrides to the config can be set from environment variables, using the `MIDGARD_` prefix. Fields in nested structs are accessed using underscores.

Examples:
* `MIDGARD_LISTEN_PORT` env variable will override `Config.ListenPort` value
* `MIDGARD_TIMESCALE_PORT` will override `Config.TimeScale.Port` value
* `MIDGARD_USD_POOLS="A,B,C"` will override the UsdPools

### Testing

```bash
docker-compose up -d pgtest
go test -p 1 ./...
```

### State Checks

A cmd that checks the state recreated by Midgard through events and the actual state stored
in the Thorchain can be run with:

```bash
go run ./cmd/statechecks config/config.json
```

### Trimming the database

Regenerating the database from height 1 can be time consuming. If there is a bug in a later point
it's possible to trim back all database tables to just before the problematic point. This is
useful to apply a bugfix quickly.

```bash
go run ./cmd/trimdb config/config.json HEIGHTORTIMESTAMP
```

### Saving & copying the database

If you'd like to do some (potentially destructive) experiments with the database, it's probably
a good idea to make a backup of it first, so you don't have to resync in case things don't go as
expected.

Consider treating unset parameters as an error when substituting.

```bash
set -u
```

Creating a backup of the `pg` instance:

```bash
# choose where to put the backup:
backup_dir=/tmp/pgbackup
# query the location of the docker volume:
pg_volume="$(docker inspect midgard_pg_1 | jq -r '.[].Mounts | .[].Source')"

# stop, backup, restart:
docker stop midgard_pg_1
sudo cp -a $pg_volume/ $backup_dir/
docker start midgard_pg_1
```

Restoring the DB from the backup:

```bash
docker stop midgard_pg_1
sudo rsync -ac --del $backup_dir/ $pg_volume/
docker start midgard_pg_1
```

Of course, you can do this with the `pg2` or `pgtest` instances too.

### Monitoring more than one chain

It is possible to rune more than one Midgard instance against different chains (e.g. main/testnet).
Create two config files (e.g. mainnet.json, testnet.json):
* set listen_port to 8080 and 8081
* edit thornode and tendermint urls
* set timescale/port to 5432 and 6432

```sh
docker-compose up --build -d pg
docker-compose up --build -d pg2
go run ./cmd/midgard tmp/mainnet.json
go run ./cmd/midgard tmp/testnet.json
```

Then you can check depths separately for them:

```bash
go run ./cmd/statechecks tmp/mainnet.json
go run ./cmd/statechecks tmp/testnet.json
```


### Gernerated files

Some GraphQL or OpenApi files are generated.

You need to install a few things once.
* For redoc-cli do `npm install` which takes dependencies form `package.json`
* For oapi-codegen do `go get github.com/deepmap/oapi-codegen/cmd/oapi-codegen`
  which will provide `$GOPATH/bin/oapi-codegen`

Then from now you can regenerate files with:

```bash
make generated
```

### Format, Lint

You can run these before submit to make sure the CI will pass:
```
gofmt -l -s -w ./
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint golangci-lint run -v
```

### Architecture

The `chain` package reads the blockchain in choronological order.
Blocks are parsed with `events` and persisted with `internal/timeseries`.
The RDBM is almost a one-to-one mapping of the *key-value entries* from the THORChain.
Aggregated values and tables are created separately in `aggregate.go`.

Package `internal/api` defines the HTTP interface. See `internal/graphql` for the query
facilities (provided by `internal/timeseries/stat`).

Blocks are “committed” with an entry in the `block_log` table, including a `block_timestamp`.
Queries give consistent [cachable] results when executed with a (time) `db.Window` within
`timeseries.LastBlock`.

## Bookmarks

Direct links:
* Testnet seed location: https://testnet.seed.thorchain.info/
* Thorchain pools at height: https://testnet.thornode.thorchain.info/thorchain/pools?height=20000
* Thorchain single pool at height: https://testnet.thornode.thorchain.info/thorchain/pool/ETH.ETH?height=20000
* Tendermint block: http://&lt;tendermint&gt;:26657/block_results?height=1000
  This is base64 wrapped, to find the readable version use `cmd/fetchblock`.

Documentation:
* Connecting to Thorchain: https://docs.thorchain.org/developers/connecting-to-thorchain
* Tendermint doc: https://docs.tendermint.com/master/rpc/#/
* Midgard doc: https://testnet.midgard.thorchain.info/v2/doc
