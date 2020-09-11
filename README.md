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
docker-compose up -d pg
```

If you don't have a THOR node to connect to use the mock.

```sh
@docker-compose up -d thormock
```

Run a local instance direct from the sources.

```sh
go run ./cmd/midgard cmd/midgard/config.json
```

Midgard populates the database with content from the blockchain.
You can see progress at <http://localhost:8080/metrics>.

Open <http://localhost:8080/v2> in your browser for the GraphQL UI.



### Testing

```bash
@docker-compose up -d thormock
go test ./...
```

Alternatively, you may omit the database tests with `go test -short ./...`.
