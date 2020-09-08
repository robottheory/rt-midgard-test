// Package graphql provides the query interface.
package graphql

import (
	"time"

	"github.com/samsarahq/thunder/graphql"
	"github.com/samsarahq/thunder/graphql/introspection"
	"github.com/samsarahq/thunder/graphql/schemabuilder"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

// stubs
var (
	lastBlock           = timeseries.LastBlock
	poolBuySwapsLookup  = stat.PoolBuySwapsLookup
	poolSellSwapsLookup = stat.PoolSellSwapsLookup
	poolGasLookup       = stat.PoolGasLookup
)

var Schema *graphql.Schema

func init() {
	builder := schemabuilder.NewSchema()
	registerQuery(builder)
	registerPool(builder)

	Schema = builder.MustBuild()

	introspection.AddIntrospectionToSchema(Schema)
}

func registerQuery(schema *schemabuilder.Schema) {
	object := schema.Query()

	object.FieldFunc("pool", func(args struct {
		Asset string
		Since *time.Time
		Until *time.Time
	}) *Pool {
		p := Pool{Asset: args.Asset}
		if args.Since != nil {
			p.window.Since = *args.Since
		}
		if args.Until != nil {
			p.window.Until = *args.Until
		} else {
			_, timestamp, _ := lastBlock()
			p.window.Until = timestamp
		}
		return &p
	})
}

type Pool struct {
	Asset  string
	window stat.Window
}

func registerPool(schema *schemabuilder.Schema) {
	object := schema.Object("Pool", Pool{})
	object.Key("asset")

	object.FieldFunc("buyStats", func(p *Pool) (*stat.PoolSwaps, error) {
		return poolBuySwapsLookup(p.Asset, p.window)
	})
	object.FieldFunc("sellStats", func(p *Pool) (*stat.PoolSwaps, error) {
		return poolSellSwapsLookup(p.Asset, p.window)
	})
	object.FieldFunc("gasStats", func(p *Pool) (stat.PoolGas, error) {
		return poolGasLookup(p.Asset, p.window)
	})
}
