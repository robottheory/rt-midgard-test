-- +migrate Up

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;


CREATE TABLE add_events (
	tx		CHAR(64) NOT NULL,
	chain		VARCHAR(16) NOT NULL,
	from_addr	VARCHAR(64) NOT NULL,
	to_addr		VARCHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	memo		TEXT NOT NULL,
	rune_E8		BIGINT NOT NULL,
	pool		VARCHAR(64) NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('add_events', 'block_timestamp');


CREATE TABLE bond_events (
	tx		CHAR(64) NOT NULL,
	chain		VARCHAR(16) NOT NULL,
	from_addr	VARCHAR(64) NOT NULL,
	to_addr		VARCHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	memo		TEXT NOT NULL,
	bound_type	VARCHAR(32) NOT NULL,
	E8		BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('bond_events', 'block_timestamp');


CREATE TABLE errata_events (
	in_tx		CHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	rune_E8		BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('errata_events', 'block_timestamp');


CREATE TABLE fee_events (
	tx		CHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	pool_deduct	BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('fee_events', 'block_timestamp');


CREATE TABLE gas_events (
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	rune_E8		BIGINT NOT NULL,
	tx_count	BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('gas_events', 'block_timestamp');


CREATE TABLE outbound_events (
	tx		CHAR(64) NOT NULL,
	chain		VARCHAR(16) NOT NULL,
	from_addr	VARCHAR(127) NOT NULL,
	to_addr		VARCHAR(127) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	memo		TEXT NOT NULL,
	in_tx		CHAR(64) NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('outbound_events', 'block_timestamp');

CREATE INDEX ON outbound_events (in_tx);


CREATE TABLE pool_events (
	asset		VARCHAR(32) NOT NULL,
	status		VARCHAR(64) NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('pool_events', 'block_timestamp');


CREATE TABLE refund_events (
	tx		CHAR(64) NOT NULL,
	chain		VARCHAR(16) NOT NULL,
	from_addr	VARCHAR(64) NOT NULL,
	to_addr		VARCHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	memo		TEXT NOT NULL,
	code		BIGINT NOT NULL,
	reason		TEXT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('refund_events', 'block_timestamp');


CREATE TABLE reserve_events (
	tx		CHAR(64) NOT NULL,
	chain		VARCHAR(16) NOT NULL,
	from_addr	VARCHAR(64) NOT NULL,
	to_addr		VARCHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	memo		TEXT NOT NULL,
	addr		VARCHAR(128) NOT NULL,
	E8		BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('reserve_events', 'block_timestamp');


CREATE TABLE rewards_events (
	bond_E8		BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('rewards_events', 'block_timestamp');

CREATE TABLE rewards_pools (
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('rewards_pools', 'block_timestamp');


CREATE TABLE stake_events (
	pool		VARCHAR(32) NOT NULL,
	asset_tx	VARCHAR(127) NOT NULL,
	asset_chain	VARCHAR(16) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	stake_units	BIGINT NOT NULL,
	rune_tx		CHAR(64) NOT NULL,
	rune_addr	CHAR(64) NOT NULL,
	rune_E8		BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('stake_events', 'block_timestamp');


CREATE TABLE swap_events (
	tx		CHAR(64) NOT NULL,
	chain		VARCHAR(16) NOT NULL,
	from_addr	VARCHAR(64) NOT NULL,
	to_addr		VARCHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	memo		TEXT NOT NULL,
	pool		VARCHAR(64) NOT NULL,
	price_target	BIGINT NOT NULL,
	trade_slip	BIGINT NOT NULL,
	liq_fee		BIGINT NOT NULL,
	liq_fee_in_rune	BIGINT NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('swap_events', 'block_timestamp');


CREATE TABLE unstake_events (
	tx		CHAR(64) NOT NULL,
	chain		VARCHAR(16) NOT NULL,
	from_addr	VARCHAR(64) NOT NULL,
	to_addr		VARCHAR(64) NOT NULL,
	asset		VARCHAR(32) NOT NULL,
	asset_E8	BIGINT NOT NULL,
	memo		TEXT NOT NULL,
	pool		VARCHAR(64) NOT NULL,
	stake_units	BIGINT NOT NULL,
	basis_points	BIGINT NOT NULL,
	asymmetry	DOUBLE PRECISION NOT NULL,
	block_timestamp	TIMESTAMP WITHOUT TIME ZONE NOT NULL
);

SELECT create_hypertable('unstake_events', 'block_timestamp');


-- +migrate Down

DROP TABLE add_events;
DROP TABLE fee_events;
DROP TABLE gas_events;
DROP TABLE outbound_events;
DROP TABLE pool_events;
DROP TABLE refund_events;
DROP TABLE reserve_events;
DROP TABLE stake_events;
DROP TABLE swap_events;
DROP TABLE unstake_events;
