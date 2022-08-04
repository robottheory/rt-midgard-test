package db

import _ "embed"

//go:embed ddl.sql
var dataDDL string

func CoreDDL() []string {
	return []string{TableCleanup("midgard"), TableCleanup("midgard_agg"), dataDDL}
}

// We have so many tables that dropping them all in one transaction makes us run out of locks
// (max_locks_per_transaction is too low). So, we drop the tables one-by-one in separate
// transactions.
func TableCleanup(schema string) string {
	return `
-- version 27

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

----------
-- Clean up

DROP SCHEMA IF EXISTS midgard_agg CASCADE;
DROP SCHEMA IF EXISTS midgard CASCADE;

-- TODO(huginn): remove after a few versions
-- Transitional, remove tables owned by us in the "public" schema. We used to work in it instead
-- of our own.
DO $$ DECLARE
	r RECORD;
BEGIN
	FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = '` + schema + `') LOOP
		EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
		COMMIT;
	END LOOP;
END $$;
`
}
