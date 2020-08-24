-- +migrate Up

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;


CREATE TABLE add_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	pool			VARCHAR(32) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('add_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE bond_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	bound_type		VARCHAR(32) NOT NULL,
	E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('bond_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE errata_events (
	in_tx			CHAR(64) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('errata_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE fee_events (
	tx			CHAR(64) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	pool_deduct		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('fee_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE gas_events (
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	tx_count		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('gas_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE new_node_events (
	thor_addr		CHAR(48) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('new_node_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE outbound_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	in_tx			CHAR(64) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('outbound_events', 'block_timestamp', chunk_time_interval => 86400000000000);

CREATE INDEX ON outbound_events USING HASH (in_tx);


CREATE TABLE pool_events (
	asset			VARCHAR(32) NOT NULL,
	status			VARCHAR(64) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('pool_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE refund_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	code			BIGINT NOT NULL,
	reason			TEXT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('refund_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE reserve_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	addr			CHAR(48) NOT NULL,
	E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('reserve_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE rewards_events (
	bond_E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('rewards_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE rewards_pools (
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('rewards_pools', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE set_ip_address_events (
	thor_addr		CHAR(44) NOT NULL,
	addr			VARCHAR(45) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('set_ip_address_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE set_node_keys_events (
	thor_addr		CHAR(44) NOT NULL,
	secp256k1_pub_key	CHAR(76) NOT NULL,
	ed25519_pub_key		CHAR(76) NOT NULL,
	validator_pub_key	CHAR(76) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('set_node_keys_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE set_version_events (
	thor_addr		CHAR(44) NOT NULL,
	version			VARCHAR(128) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('set_version_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE slash_amounts (
	pool			VARCHAR(32) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('slash_amounts', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE stake_events (
	pool			VARCHAR(32) NOT NULL,
	asset_tx		VARCHAR(64) NOT NULL,
	asset_chain		VARCHAR(8) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	stake_units		BIGINT NOT NULL,
	rune_tx			CHAR(64) NOT NULL,
	rune_addr		CHAR(48) NOT NULL,
	rune_E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('stake_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE swap_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	pool			VARCHAR(32) NOT NULL,
	price_target		BIGINT NOT NULL,
	trade_slip		BIGINT NOT NULL,
	liq_fee			BIGINT NOT NULL,
	liq_fee_in_rune		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('swap_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE unstake_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(32) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	pool			VARCHAR(32) NOT NULL,
	stake_units		BIGINT NOT NULL,
	basis_points		BIGINT NOT NULL,
	asymmetry		DOUBLE PRECISION NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('unstake_events', 'block_timestamp', chunk_time_interval => 86400000000000);


-- +migrate Down

DROP TABLE add_events;
DROP TABLE bond_events;
DROP TABLE errata_events;
DROP TABLE fee_events;
DROP TABLE gas_events;
DROP TABLE new_node_events;
DROP TABLE outbound_events;
DROP TABLE pool_events;
DROP TABLE refund_events;
DROP TABLE reserve_events;
DROP TABLE rewards_events;
DROP TABLE rewards_pools;
DROP TABLE set_ip_address_events;
DROP TABLE set_node_keys_events;
DROP TABLE set_version_events;
DROP TABLE slash_amounts;
DROP TABLE stake_events;
DROP TABLE swap_events;
DROP TABLE unstake_events;