package db

func AggregatesDdl() string {
	return `
-- version 1

DROP SCHEMA IF EXISTS midgard_agg CASCADE;
CREATE SCHEMA midgard_agg;

`
}
