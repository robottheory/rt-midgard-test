package db

func Ddl() string {
	return `
-- version 12

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
    FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tableowner='midgard') LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
    END LOOP;
END $$;

----------
-- Fresh start

CREATE SCHEMA midgard;

-- Check that the newly created schema is the one we are going to work with.
-- If someone uses a non-standard set up, like using a different postgres user name, it's better
-- to abort at this point and let them know that it's not going to work.
DO $$ BEGIN
	ASSERT (SELECT current_schema()) = 'midgard', 'current_schema() is not midgard';
END $$;


CREATE TABLE constants (
  key VARCHAR(30) NOT NULL,
  value BYTEA NOT NULL,
  PRIMARY KEY (key)
);

CREATE TABLE block_log (
	height			BIGINT NOT NULL,
	timestamp		BIGINT NOT NULL,
	hash			BYTEA NOT NULL,
	agg_state		BYTEA,
	PRIMARY KEY (height)
);

CREATE INDEX ON block_log (timestamp DESC);


-- For hypertables with an integer 'time' dimension (as opposed to TIMESTAMPTZ),
-- TimescaleDB requires an 'integer_now' function to be set to use continuous aggregates.
-- We use the following function, 'current_nano', as the 'integer_now' function
-- for all of our hypertables.
--
-- This function is only comes into play if one uses TimescaleDB's automatic refresh policies
-- for continuous aggregates. As we trigger refreshes directly from Midgard, what this
-- function does is basically irrelevant, so we choose to return the most directly
-- corresponding notion of 'now'.
--
-- An alternative approach would be to get the latest block timestamp from 'block_log' or some
-- other table and use TimescaleDB's automatic refresh policies. (The downside is that it gets
-- harder to control, if for example we want to suspend refreshing, etc.)
CREATE OR REPLACE FUNCTION current_nano() RETURNS BIGINT
LANGUAGE SQL STABLE AS $$
    SELECT CAST(1000000000 * EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) AS BIGINT)
$$;

CREATE PROCEDURE setup_hypertable(t regclass)
LANGUAGE SQL
AS $$
    SELECT create_hypertable(t, 'block_timestamp', chunk_time_interval => 86400000000000);
    SELECT set_integer_now_func(t, 'current_nano');
$$;


-- The standard PostgreSQL 'date_trunc(field, timestamp)' function,
--  but takes and returns 'nanos from epoch'
CREATE OR REPLACE FUNCTION nano_trunc(field TEXT, ts BIGINT) RETURNS BIGINT
LANGUAGE SQL IMMUTABLE AS $$
    SELECT CAST(1000000000 * EXTRACT(EPOCH FROM date_trunc(field, to_timestamp(ts / 1000000000))) AS BIGINT)
$$;


-- Sparse table for depths.
-- Only those height/pool pairs are filled where there is a change.
-- For missing values, use the latest existing height for a pool.
-- Asset and Rune are filled together, it's not needed to look back for them separately.
CREATE TABLE block_pool_depths (
	pool				VARCHAR(60) NOT NULL,
	asset_E8			BIGINT NOT NULL,
	rune_E8				BIGINT NOT NULL,
	synth_E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('block_pool_depths');
CREATE INDEX ON block_pool_depths (pool, block_timestamp DESC);


CREATE TABLE active_vault_events (
	add_asgard_addr		VARCHAR(90) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('active_vault_events');


CREATE TABLE add_events (
	tx  			VARCHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		VARCHAR(90) NOT NULL,
	to_addr			VARCHAR(90) NOT NULL,
	asset			VARCHAR(60),
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	pool			VARCHAR(60) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('add_events');


CREATE TABLE asgard_fund_yggdrasil_events (
	tx	    		VARCHAR(64) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	vault_key		VARCHAR(90) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('asgard_fund_yggdrasil_events');


CREATE TABLE bond_events (
	tx		    	VARCHAR(64) NOT NULL,
	chain			VARCHAR(8),
	from_addr		VARCHAR(90),
	to_addr			VARCHAR(90),
	asset			VARCHAR(60),
	asset_E8		BIGINT NOT NULL,
	memo			TEXT,
	bond_type		VARCHAR(32) NOT NULL,
	E8			    BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('bond_events');


CREATE TABLE errata_events (
	in_tx			VARCHAR(64) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('errata_events');


CREATE TABLE fee_events (
	tx			VARCHAR(64) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	pool_deduct		BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('fee_events');
CREATE INDEX fee_events_tx_idx ON fee_events (tx);


CREATE TABLE gas_events (
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	tx_count		BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('gas_events');


CREATE TABLE inactive_vault_events (
	add_asgard_addr		VARCHAR(90) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('inactive_vault_events');


CREATE TABLE set_mimir_events (
	key			        VARCHAR(63) NOT NULL,
	value			    VARCHAR(127) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('set_mimir_events');


CREATE TABLE message_events (
	from_addr		    VARCHAR(90) NOT NULL,
	action			    VARCHAR(31) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('message_events');


CREATE TABLE new_node_events (
	node_addr		    VARCHAR(48) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('new_node_events');


CREATE TABLE outbound_events (
	tx			    VARCHAR(64),
	chain			VARCHAR(8) NOT NULL,
	from_addr		VARCHAR(90) NOT NULL,
	to_addr			VARCHAR(90) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	in_tx			VARCHAR(64) NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('outbound_events');
CREATE INDEX outbound_events_in_tx_idx ON outbound_events (in_tx);

CREATE TABLE pool_events (
	asset			VARCHAR(60) NOT NULL,
	status			VARCHAR(64) NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('pool_events');


CREATE TABLE refund_events (
	tx			    VARCHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		VARCHAR(90) NOT NULL,
	to_addr			VARCHAR(90) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	asset_2nd		VARCHAR(60),
	asset_2nd_E8	BIGINT NOT NULL,
	memo			TEXT,
	code			BIGINT NOT NULL,
	reason			TEXT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('refund_events');


CREATE TABLE reserve_events (
	tx			    VARCHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		VARCHAR(90) NOT NULL,
	to_addr			VARCHAR(90) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	addr			VARCHAR(48) NOT NULL,
	E8			    BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('reserve_events');


CREATE TABLE rewards_events (
	bond_E8			    BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('rewards_events');


CREATE TABLE rewards_event_entries (
	pool			    VARCHAR(60) NOT NULL,
	rune_E8			    BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('rewards_event_entries');


CREATE TABLE set_ip_address_events (
	node_addr		    VARCHAR(44) NOT NULL,
	ip_addr			    VARCHAR(45) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('set_ip_address_events');


CREATE TABLE set_node_keys_events (
	node_addr   		VARCHAR(44) NOT NULL,
	secp256k1	    	VARCHAR(90) NOT NULL,
	ed25519			    VARCHAR(90) NOT NULL,
	validator_consensus	VARCHAR(90) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('set_node_keys_events');


CREATE TABLE set_version_events (
	node_addr		    VARCHAR(44) NOT NULL,
	version			    VARCHAR(127) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('set_version_events');


CREATE TABLE slash_amounts (
	pool			    VARCHAR(60) NOT NULL,
	asset			    VARCHAR(60) NOT NULL,
	asset_E8		    BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('slash_amounts');


CREATE TABLE stake_events (
	pool			VARCHAR(60) NOT NULL,
	asset_tx		VARCHAR(64),
	asset_chain		VARCHAR(8),
	asset_addr		VARCHAR(90),
	asset_E8		BIGINT NOT NULL,
	stake_units		BIGINT NOT NULL,
	rune_tx			VARCHAR(64),
	rune_addr		VARCHAR(90),
	rune_E8			BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('stake_events');


CREATE TABLE pending_liquidity_events (
	pool			VARCHAR(60) NOT NULL,
	asset_tx		VARCHAR(64),
	asset_chain		VARCHAR(8),
	asset_addr		VARCHAR(90),
	asset_E8		BIGINT NOT NULL,
	rune_tx			VARCHAR(64),
	rune_addr		VARCHAR(90),
	rune_E8			BIGINT NOT NULL,
	pending_type	VARCHAR(10) NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('pending_liquidity_events');

CREATE TABLE swap_events (
	tx			        VARCHAR(64) NOT NULL,
	chain			    VARCHAR(8) NOT NULL,
	from_addr		    VARCHAR(90) NOT NULL,
	to_addr			    VARCHAR(90) NOT NULL,
	from_asset		    VARCHAR(60) NOT NULL,
	from_E8			    BIGINT NOT NULL,
	to_asset		    VARCHAR(60) NOT NULL,
	to_E8			    BIGINT NOT NULL,
	memo			    TEXT NOT NULL,
	pool			    VARCHAR(60) NOT NULL,
	to_E8_min		    BIGINT NOT NULL,
	swap_slip_BP	    BIGINT NOT NULL,
	liq_fee_E8		    BIGINT NOT NULL,
	liq_fee_in_rune_E8	BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('swap_events');


CREATE TABLE switch_events (
	from_addr		    VARCHAR(90) NOT NULL,
	to_addr			    VARCHAR(90) NOT NULL,
	burn_asset		    VARCHAR(60) NOT NULL,
	burn_E8			    BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('switch_events');


CREATE TABLE transfer_events (
	from_addr		VARCHAR(90) NOT NULL,
	to_addr			VARCHAR(90) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	amount_E8		BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('transfer_events');


CREATE TABLE unstake_events (
	tx			    VARCHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		VARCHAR(90) NOT NULL,
	to_addr			VARCHAR(90) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	emit_asset_E8	BIGINT NOT NULL,
	emit_rune_E8	BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	pool			VARCHAR(60) NOT NULL,
	stake_units		BIGINT NOT NULL,
	basis_points	BIGINT NOT NULL,
	asymmetry		DOUBLE PRECISION NOT NULL,
	imp_loss_protection_E8 BIGINT NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('unstake_events');


CREATE TABLE update_node_account_status_events (
	node_addr		VARCHAR(90) NOT NULL,
	former			VARCHAR(31) NOT NULL,
	current			VARCHAR(31) NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('update_node_account_status_events');


CREATE TABLE validator_request_leave_events (
	tx			    VARCHAR(64) NOT NULL,
	from_addr		VARCHAR(90) NOT NULL,
	node_addr		VARCHAR(90) NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('validator_request_leave_events');

CREATE TABLE pool_balance_change_events (
	asset			VARCHAR(60) NOT NULL,
	rune_amt        BIGINT NOT NULL,
	rune_add        BOOLEAN NOT NULL,
	asset_amt       BIGINT NOT NULL,
	asset_add       BOOLEAN NOT NULL,
	reason          VARCHAR(100) NOT NULL,
	block_timestamp	BIGINT NOT NULL
);

CALL setup_hypertable('pool_balance_change_events');

CREATE TABLE thorname_change_events (
	name				VARCHAR(30) NOT NULL,
	chain				VARCHAR(8) NOT NULL,
	address				VARCHAR(90) NOT NULL,
	registration_fee_e8 BIGINT NOT NULL,
	fund_amount_e8		BIGINT NOT NULL,
	expire				BIGINT NOT NULL,
	owner				VARCHAR(90) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

CALL setup_hypertable('thorname_change_events');
CREATE INDEX ON thorname_change_events (name DESC);
CREATE INDEX ON thorname_change_events (address DESC);
`
}
