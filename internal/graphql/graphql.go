// Package graphql provides the query interface.
package graphql

import (
	"errors"
	"fmt"
	"time"

	"github.com/samsarahq/thunder/graphql"
	"github.com/samsarahq/thunder/graphql/introspection"
	"github.com/samsarahq/thunder/graphql/schemabuilder"

	"gitlab.com/thorchain/midgard/internal/timeseries"
	"gitlab.com/thorchain/midgard/internal/timeseries/stat"
)

// stubs
var (
	lastBlock                 = timeseries.LastBlock
	allPoolStakesAddrLookup   = stat.AllPoolStakesAddrLookup
	poolSwapsFromRuneLookup        = stat.PoolSwapsFromRuneLookup
	poolSwapsFromRuneBucketsLookup = stat.PoolSwapsFromRuneBucketsLookup
	poolGasLookup             = stat.PoolGasLookup

	poolSwapsToRuneLookup        = stat.PoolSwapsToRuneLookup
	poolSwapsToRuneBucketsLookup = stat.PoolSwapsToRuneBucketsLookup
	poolStakesBucketsLookup    = stat.PoolStakesBucketsLookup
	poolStakesLookup           = stat.PoolStakesLookup
	stakesAddrLookup           = stat.StakesAddrLookup
)

var Schema *graphql.Schema

func init() {
	builder := schemabuilder.NewSchema()
	registerQuery(builder)
	registerPool(builder)
	registerStaker(builder)

	Schema = builder.MustBuild()

	introspection.AddIntrospectionToSchema(Schema)
}

func registerQuery(schema *schemabuilder.Schema) {
	object := schema.Query()

	object.FieldFunc("pool", func(args struct {
		Asset      string
		Since      *time.Time
		Until      *time.Time
		BucketSize *string
	}) (*Pool, error) {
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

		if args.BucketSize != nil {
			var err error
			p.bucketSize, err = time.ParseDuration(*args.BucketSize)
			if err != nil {
				return nil, fmt.Errorf("malformed bucket size: %w", err)
			}
		} else {
			p.bucketSize = time.Hour
		}

		return &p, nil
	})

	object.FieldFunc("staker", func(args struct {
		Addr  string
		Since *time.Time
		Until *time.Time
	}) (*Staker, error) {
		r := Staker{Addr: args.Addr}
		if args.Since != nil {
			r.window.Since = *args.Since
		}
		if args.Until != nil {
			r.window.Until = *args.Until
		} else {
			_, timestamp, _ := lastBlock()
			r.window.Until = timestamp
		}

		stakes, err := stakesAddrLookup(r.Addr, r.window)
		if err != nil {
			return nil, err
		}
		if stakes.Last.IsZero() {
			return nil, errors.New("staker not found—no stakes for address")
		}
		r.Stakes = *stakes

		return &r, nil
	})
}

type Pool struct {
	Asset      string
	window     stat.Window
	bucketSize time.Duration
}

func registerPool(schema *schemabuilder.Schema) {
	object := schema.Object("Pool", Pool{})
	object.Key("asset")

	object.FieldFunc("stakeStats", func(p *Pool) (*stat.PoolStakes, error) {
		return poolStakesLookup(p.Asset, p.window)
	})
	object.FieldFunc("stakesBuckets", func(p *Pool) ([]stat.PoolStakes, error) {
		return poolStakesBucketsLookup(p.Asset, p.bucketSize, p.window)
	})
	object.FieldFunc("swapsFromRuneStats", func(p *Pool) (*stat.PoolSwaps, error) {
		return poolSwapsFromRuneLookup(p.Asset, p.window)
	})
	object.FieldFunc("swapsFromRuneBuckets", func(p *Pool) ([]*stat.PoolSwaps, error) {
		return poolSwapsFromRuneBucketsLookup(p.Asset, p.bucketSize, p.window)
	})
	object.FieldFunc("swapsToRuneStats", func(p *Pool) (*stat.PoolSwaps, error) {
		return poolSwapsToRuneLookup(p.Asset, p.window)
	})
	object.FieldFunc("swapsToRuneBuckets", func(p *Pool) ([]*stat.PoolSwaps, error) {
		return poolSwapsToRuneBucketsLookup(p.Asset, p.bucketSize, p.window)
	})
	object.FieldFunc("gasStats", func(p *Pool) (*stat.PoolGas, error) {
		return poolGasLookup(p.Asset, p.window)
	})
}

type Staker struct {
	Addr   string
	window stat.Window

	stat.Stakes
}

func registerStaker(schema *schemabuilder.Schema) {
	object := schema.Object("Staker", Staker{})
	object.Key("addr")

	object.FieldFunc("stakeStats", func(r *Staker) ([]stat.PoolStakes, error) {
		return allPoolStakesAddrLookup(r.Addr, r.window)
	})
}
