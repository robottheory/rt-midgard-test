# Midgard config

If you have trouble managing your config files this doc might help.

## Composing many configs

The tmp folder is gitignored, copy all examples over and edit to your taste:

```
$ cp -r ./config/examples ./tmp/config
```

Then you can run Midgard and swap individual aspects of the config from command line.
Later configs overwrite values from earlier ones.

```
$ go run ./cmd/midgard tmp/config/base.json:tmp/config/pg-1.json:tmp/config/bs-1.json:tmp/config/mainnet-tc.json
```

## Overwrite single values with environment variables

Overrides to the config can be set from environment variables, using the `MIDGARD_` prefix.
Fields in nested structs are accessed using underscores.

Examples:
* `MIDGARD_LISTEN_PORT` env variable will override `Config.ListenPort` value
* `MIDGARD_TIMESCALE_PORT` will override `Config.TimeScale.Port` value
* `MIDGARD_USD_POOLS="A,B,C"` will override the UsdPools
