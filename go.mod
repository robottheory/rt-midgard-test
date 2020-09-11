module gitlab.com/thorchain/midgard

go 1.13

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/cosmos/cosmos-sdk v0.38.3
	github.com/deepmap/oapi-codegen v1.3.6
	github.com/ethereum/go-ethereum v1.9.14
	github.com/getkin/kin-openapi v0.2.0
	github.com/google/go-cmp v0.5.0 // indirect
	github.com/graphql-go/graphql v0.7.9 // indirect
	github.com/huandu/go-sqlbuilder v1.7.0
	github.com/jackc/pgx/v4 v4.8.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/julienschmidt/httprouter v1.2.0
	github.com/labstack/echo/v4 v4.1.11
	github.com/lib/pq v1.3.0
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/ginkgo v1.10.3 // indirect
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/openlyinc/pointy v1.1.2
	github.com/pascaldekloe/metrics v1.2.0
	github.com/pascaldekloe/sqltest v0.1.0
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/rogpeppe/go-internal v1.6.0 // indirect
	github.com/rs/zerolog v1.17.2
	github.com/rubenv/sql-migrate v0.0.0-20200616145509-8d140a17f351
	github.com/samsarahq/go v0.0.0-20191220233105-8077c9fbaed5 // indirect
	github.com/samsarahq/thunder v0.5.0
	github.com/spf13/viper v1.6.3
	github.com/tendermint/tendermint v0.33.4
	github.com/ziflex/lecho/v2 v2.0.0
	google.golang.org/genproto v0.0.0-20191007204434-a023cd5227bd // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
)

replace github.com/tendermint/go-amino => github.com/binance-chain/bnc-go-amino v0.14.1-binance.1

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
