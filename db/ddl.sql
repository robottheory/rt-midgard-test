CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

CREATE TABLE block_log (
	height			BIGINT NOT NULL,
	timestamp		BIGINT NOT NULL,
	hash			BYTEA NOT NULL,
	agg_state		BYTEA,
	PRIMARY KEY (height)
);


CREATE TABLE add_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	pool			VARCHAR(60) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('add_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE bond_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	bound_type		VARCHAR(32) NOT NULL,
	E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('bond_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE errata_events (
	in_tx			CHAR(64) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('errata_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE fee_events (
	tx			CHAR(64) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	pool_deduct		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('fee_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE gas_events (
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	rune_E8			BIGINT NOT NULL,
	tx_count		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('gas_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE new_node_events (
	node_addr		CHAR(48) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('new_node_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE outbound_events (
	tx			CHAR(64),
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	in_tx			CHAR(64) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('outbound_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE pool_events (
	asset			VARCHAR(60) NOT NULL,
	status			VARCHAR(64) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('pool_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE refund_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	asset_2nd		VARCHAR(60),
	asset_2nd_E8		BIGINT NOT NULL,
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
	asset			VARCHAR(60) NOT NULL,
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


CREATE TABLE rewards_event_entries (
	pool			VARCHAR(60) NOT NULL,
	rune_E8			BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('rewards_event_entries', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE set_ip_address_events (
	node_addr		CHAR(44) NOT NULL,
	ip_addr			VARCHAR(45) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('set_ip_address_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE set_node_keys_events (
	node_addr		CHAR(44) NOT NULL,
	secp256k1		CHAR(76) NOT NULL,
	ed25519			CHAR(76) NOT NULL,
	validator_consensus	CHAR(76) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('set_node_keys_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE set_version_events (
	node_addr		CHAR(44) NOT NULL,
	version			VARCHAR(127) NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('set_version_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE slash_amounts (
	pool			VARCHAR(60) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('slash_amounts', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE stake_events (
	pool			VARCHAR(60) NOT NULL,
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
	from_asset		VARCHAR(60) NOT NULL,
	from_E8			BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	pool			VARCHAR(60) NOT NULL,
	to_E8_min		BIGINT NOT NULL,
	trade_slip_BP		BIGINT NOT NULL,
	liq_fee_E8		BIGINT NOT NULL,
	liq_fee_in_rune_E8	BIGINT NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('swap_events', 'block_timestamp', chunk_time_interval => 86400000000000);


CREATE TABLE unstake_events (
	tx			CHAR(64) NOT NULL,
	chain			VARCHAR(8) NOT NULL,
	from_addr		CHAR(48) NOT NULL,
	to_addr			CHAR(48) NOT NULL,
	asset			VARCHAR(60) NOT NULL,
	asset_E8		BIGINT NOT NULL,
	memo			TEXT NOT NULL,
	pool			VARCHAR(60) NOT NULL,
	stake_units		BIGINT NOT NULL,
	basis_points		BIGINT NOT NULL,
	asymmetry		DOUBLE PRECISION NOT NULL,
	block_timestamp		BIGINT NOT NULL
);

SELECT create_hypertable('unstake_events', 'block_timestamp', chunk_time_interval => 86400000000000);
