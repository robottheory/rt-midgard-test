[![pipeline status](https://gitlab.com/thorchain/midgard/badges/master/pipeline.svg)](https://gitlab.com/thorchain/midgard/commits/master)


****

> **Mirror**
>
> This repo mirrors from THORChain Gitlab to Github. 
> To contribute, please contact the team and commit to the Gitlab repo:
>
> https://gitlab.com/thorchain/midgard

****


# Midgard API 

Midgard is a layer 2 REST API that provides front-end consumers with semi real-time rolled up data and analytics of the THORChain network. Most requests to the network will come through Midgard. This daemon is here to keep the chain itself from fielding large quantities of requests. You can think of it as a “read-only slave” to the chain. This keeps the resources of the network focused on processing transactions.



### Run Midgard

The daemon needs PostgreSQL with the TimeScale extension.

```sh
docker-compose up --build -d pg
```

Now you can launch a local instance directly from the sources.

```sh
go run ./cmd/midgard cmd/midgard/config.json
```

`cmd/midgard/config.json` asumes you are running a ThorNode on localhost. If that is not the case or if you want to develop against a specific network, you may want to go to the [network's seed url](https://docs.thorchain.org/developers/connecting-to-thorchain) and pick a node ip from there to replace the host in `tendermint_url` and `thornode_url` with that ip.

Midgard populates the database with content from the blockchain.
Progress is traceable with the Prometheus Metrics propagated on
<http://localhost:8080/debug/metrics>, specifically the measurements
`midgard_chain_cursor_height` v.s. `midgard_chain_height`.

Open <http://localhost:8080/v2> in your browser for the GraphQL UI. ✨

### Config

Configuration is loaded from a `.json` file. Default is in `cmd/midgard/config.json`.

Overrides to the config can be set from environment variables, using the `MIDGARD_` prefix. Fields in nested structs are accessed using underscores.

Examples:
* `MIDGARD_LISTEN_PORT` env variable will override `Config.ListenPort` value
* `MIDGARD_TIMESCALE_PORT` env variable will override `Config.TimeScale.Port` value

### Testing

```bash
docker-compose up -d pgtest
go test -p 1 ./...
```

### State Checks
A cmd that checks the state recreated by Midgard through events and the actual state stored
in the Thorchain can be run with:

```bash
go run cmd/state/main.go cmd/midgard/config.json
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
