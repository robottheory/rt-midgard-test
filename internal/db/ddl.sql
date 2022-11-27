-- version 26

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

----------
-- Clean up

DROP SCHEMA IF EXISTS midgard_agg CASCADE;
DROP SCHEMA IF EXISTS midgard CASCADE;


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
                           key TEXT NOT NULL,
                           value BYTEA NOT NULL,
                           PRIMARY KEY (key)
);

CREATE TABLE block_log (
                           height          BIGINT NOT NULL,
                           timestamp       BIGINT NOT NULL,
                           hash            BYTEA NOT NULL,
                           agg_state       BYTEA,
                           PRIMARY KEY (height),
                           UNIQUE (timestamp)
);



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


-- Various time/nano/height related helper functions.
-- We don't rely on these, but they are very useful during development and debugging.
CREATE FUNCTION ts_nano(t timestamptz) RETURNS bigint
    LANGUAGE SQL IMMUTABLE AS $$
SELECT CAST(1000000000 * EXTRACT(EPOCH FROM t) AS bigint)
           $$;

CREATE FUNCTION nano_ts(t bigint) RETURNS timestamptz
    LANGUAGE SQL IMMUTABLE AS $$
SELECT to_timestamp(t/1e9);
$$;

CREATE FUNCTION height_nano(h bigint) RETURNS bigint
    LANGUAGE SQL STABLE AS $$
SELECT timestamp FROM midgard.block_log WHERE height = h;
$$;

-- Sparse table for depths.
-- Only those height/pool pairs are filled where there is a change.
-- For missing values, use the latest existing height for a pool.
-- Asset and Rune are filled together, it's not needed to look back for them separately.
CREATE TABLE block_pool_depths (
                                   pool                TEXT NOT NULL,
                                   asset_e8            BIGINT NOT NULL,
                                   rune_e8             BIGINT NOT NULL,
                                   synth_e8            BIGINT NOT NULL,
                                   price				REAL NOT NULL,
                                   priceusd			REAL NOT NULL,
                                   units 				BIGINT NOT NULL,
                                   block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('block_pool_depths');
CREATE INDEX ON block_pool_depths (pool, block_timestamp DESC);


CREATE TABLE active_vault_events (
                                     add_asgard_addr     TEXT NOT NULL,
                                     block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('active_vault_events');


CREATE TABLE add_events (
                            tx              TEXT NOT NULL,
                            chain           TEXT NOT NULL,
                            from_addr       TEXT NOT NULL,
                            to_addr         TEXT NOT NULL,
                            asset           TEXT,
                            asset_e8        BIGINT NOT NULL,
                            memo            TEXT NOT NULL,
                            rune_e8         BIGINT NOT NULL,
                            pool            TEXT NOT NULL,
                            block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('add_events');


CREATE TABLE asgard_fund_yggdrasil_events (
                                              tx              TEXT NOT NULL,
                                              asset           TEXT NOT NULL,
                                              asset_e8        BIGINT NOT NULL,
                                              vault_key       TEXT NOT NULL,
                                              block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('asgard_fund_yggdrasil_events');


CREATE TABLE bond_events (
                             tx              TEXT NOT NULL,
                             chain           TEXT,
                             from_addr       TEXT,
                             to_addr         TEXT,
                             asset           TEXT,
                             asset_e8        BIGINT NOT NULL,
                             memo            TEXT,
                             bond_type       TEXT NOT NULL,
                             e8              BIGINT NOT NULL,
                             block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('bond_events');


CREATE TABLE errata_events (
                               in_tx           TEXT NOT NULL,
                               asset           TEXT NOT NULL,
                               asset_e8        BIGINT NOT NULL,
                               rune_e8         BIGINT NOT NULL,
                               block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('errata_events');


CREATE TABLE fee_events (
                            tx              TEXT NOT NULL,
                            asset           TEXT NOT NULL,
                            asset_e8        BIGINT NOT NULL,
                            pool_deduct     BIGINT NOT NULL,
                            block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('fee_events');
CREATE INDEX fee_events_tx_idx ON fee_events (tx);


CREATE TABLE gas_events (
                            asset           TEXT NOT NULL,
                            asset_e8        BIGINT NOT NULL,
                            rune_e8         BIGINT NOT NULL,
                            tx_count        BIGINT NOT NULL,
                            block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('gas_events');


CREATE TABLE inactive_vault_events (
                                       add_asgard_addr     TEXT NOT NULL,
                                       block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('inactive_vault_events');


CREATE TABLE set_mimir_events (
                                  key                 TEXT NOT NULL,
                                  value               TEXT NOT NULL,
                                  block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('set_mimir_events');


CREATE TABLE message_events (
                                from_addr           TEXT NOT NULL,
                                action              TEXT NOT NULL,
                                block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('message_events');


CREATE TABLE new_node_events (
                                 node_addr           TEXT NOT NULL,
                                 block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('new_node_events');


CREATE TABLE outbound_events (
                                 tx              TEXT,
                                 chain           TEXT NOT NULL,
                                 from_addr       TEXT NOT NULL,
                                 to_addr         TEXT NOT NULL,
                                 asset           TEXT NOT NULL,
                                 asset_e8        BIGINT NOT NULL,
                                 memo            TEXT NOT NULL,
                                 in_tx           TEXT NOT NULL,
                                 block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('outbound_events');
CREATE INDEX outbound_events_in_tx_idx ON outbound_events (in_tx);

CREATE TABLE pool_events (
                             asset           TEXT NOT NULL,
                             status          TEXT NOT NULL,
                             block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('pool_events');


CREATE TABLE refund_events (
                               tx              TEXT NOT NULL,
                               chain           TEXT NOT NULL,
                               from_addr       TEXT NOT NULL,
                               to_addr         TEXT NOT NULL,
                               asset           TEXT NOT NULL,
                               asset_e8        BIGINT NOT NULL,
                               asset_2nd       TEXT,
                               asset_2nd_e8    BIGINT NOT NULL,
                               memo            TEXT,
                               code            BIGINT NOT NULL,
                               reason          TEXT NOT NULL,
                               block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('refund_events');


CREATE TABLE reserve_events (
                                tx              TEXT NOT NULL,
                                chain           TEXT NOT NULL,
                                from_addr       TEXT NOT NULL,
                                to_addr         TEXT NOT NULL,
                                asset           TEXT NOT NULL,
                                asset_e8        BIGINT NOT NULL,
                                memo            TEXT NOT NULL,
                                addr            TEXT NOT NULL,
                                e8              BIGINT NOT NULL,
                                block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('reserve_events');


CREATE TABLE rewards_events (
                                bond_e8         BIGINT NOT NULL,
                                block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('rewards_events');


CREATE TABLE rewards_event_entries (
                                       pool                TEXT NOT NULL,
                                       rune_e8             BIGINT NOT NULL,
                                       block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('rewards_event_entries');


CREATE TABLE set_ip_address_events (
                                       node_addr           TEXT NOT NULL,
                                       ip_addr             TEXT NOT NULL,
                                       block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('set_ip_address_events');


CREATE TABLE set_node_keys_events (
                                      node_addr           TEXT NOT NULL,
                                      secp256k1           TEXT NOT NULL,
                                      ed25519             TEXT NOT NULL,
                                      validator_consensus TEXT NOT NULL,
                                      block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('set_node_keys_events');


CREATE TABLE set_version_events (
                                    node_addr           TEXT NOT NULL,
                                    version             TEXT NOT NULL,
                                    block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('set_version_events');


CREATE TABLE slash_amounts (
                               pool                TEXT NOT NULL,
                               asset               TEXT NOT NULL,
                               asset_e8            BIGINT NOT NULL,
                               block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('slash_amounts');


CREATE TABLE stake_events (
                              pool               TEXT NOT NULL,
                              asset_tx           TEXT,
                              asset_chain        TEXT,
                              asset_addr         TEXT,
                              asset_e8           BIGINT NOT NULL,
                              stake_units        BIGINT NOT NULL,
                              rune_tx            TEXT,
                              rune_addr          TEXT,
                              rune_e8            BIGINT NOT NULL,
                              _asset_in_rune_e8  BIGINT NOT NULL,
                              block_timestamp    BIGINT NOT NULL
);

CALL setup_hypertable('stake_events');


CREATE TABLE pending_liquidity_events (
                                          pool            TEXT NOT NULL,
                                          asset_tx        TEXT,
                                          asset_chain     TEXT,
                                          asset_addr      TEXT,
                                          asset_e8        BIGINT NOT NULL,
                                          rune_tx         TEXT,
                                          rune_addr       TEXT,
                                          rune_e8         BIGINT NOT NULL,
                                          pending_type    TEXT NOT NULL,
                                          block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('pending_liquidity_events');

CREATE TABLE swap_events (
                             tx                  TEXT NOT NULL,
                             chain               TEXT NOT NULL,
                             from_addr           TEXT NOT NULL,
                             to_addr             TEXT NOT NULL,
                             from_asset          TEXT NOT NULL,
                             from_e8             BIGINT NOT NULL,
                             to_asset            TEXT NOT NULL,
                             to_e8               BIGINT NOT NULL,
                             memo                TEXT NOT NULL,
                             pool                TEXT NOT NULL,
                             to_e8_min           BIGINT NOT NULL,
                             swap_slip_bp        BIGINT NOT NULL,
                             liq_fee_e8          BIGINT NOT NULL,
                             liq_fee_in_rune_e8  BIGINT NOT NULL,
                             _direction          SMALLINT NOT NULL,  -- 0=RuneToAsset 1=AssetToRune 2=RuneToSynth 3=SynthToRune
                             block_timestamp     BIGINT NOT NULL,
                             priceusd			 REAL NOT NULL
);

CALL setup_hypertable('swap_events');


CREATE TABLE switch_events (
                               tx                  TEXT,
                               from_addr           TEXT NOT NULL,
                               to_addr             TEXT NOT NULL,
                               burn_asset          TEXT NOT NULL,
                               burn_e8             BIGINT NOT NULL,
                               mint_e8             BIGINT NOT NULL,
                               block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('switch_events');


CREATE TABLE transfer_events (
                                 from_addr       TEXT NOT NULL,
                                 to_addr         TEXT NOT NULL,
                                 asset           TEXT NOT NULL,
                                 amount_e8       BIGINT NOT NULL,
                                 block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('transfer_events');


CREATE TABLE unstake_events (
                                tx                      TEXT NOT NULL,
                                chain                   TEXT NOT NULL,
                                from_addr               TEXT NOT NULL,
                                to_addr                 TEXT NOT NULL,
                                asset                   TEXT NOT NULL,
                                asset_e8                BIGINT NOT NULL,
                                emit_asset_e8           BIGINT NOT NULL,
                                emit_rune_e8            BIGINT NOT NULL,
                                memo                    TEXT NOT NULL,
                                pool                    TEXT NOT NULL,
                                stake_units             BIGINT NOT NULL,
                                basis_points            BIGINT NOT NULL,
                                asymmetry               DOUBLE PRECISION NOT NULL,
                                imp_loss_protection_e8  BIGINT NOT NULL,
                                _emit_asset_in_rune_e8  BIGINT NOT NULL,
                                block_timestamp         BIGINT  NOT NULL
);

CALL setup_hypertable('unstake_events');


CREATE TABLE update_node_account_status_events (
                                                   node_addr       TEXT NOT NULL,
                                                   former          TEXT NOT NULL,
                                                   current         TEXT NOT NULL,
                                                   block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('update_node_account_status_events');


CREATE TABLE validator_request_leave_events (
                                                tx              TEXT NOT NULL,
                                                from_addr       TEXT NOT NULL,
                                                node_addr       TEXT NOT NULL,
                                                block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('validator_request_leave_events');

CREATE TABLE pool_balance_change_events (
                                            asset           TEXT NOT NULL,
                                            rune_amt        BIGINT NOT NULL,
                                            rune_add        BOOLEAN NOT NULL,
                                            asset_amt       BIGINT NOT NULL,
                                            asset_add       BOOLEAN NOT NULL,
                                            reason          TEXT NOT NULL,
                                            block_timestamp BIGINT NOT NULL
);

CALL setup_hypertable('pool_balance_change_events');

CREATE TABLE thorname_change_events (
                                        name                TEXT NOT NULL,
                                        chain               TEXT NOT NULL,
                                        address             TEXT NOT NULL,
                                        registration_fee_e8 BIGINT NOT NULL,
                                        fund_amount_e8      BIGINT NOT NULL,
                                        expire              BIGINT NOT NULL,
                                        owner               TEXT,
                                        block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('thorname_change_events');
CREATE INDEX ON thorname_change_events (name DESC);
CREATE INDEX ON thorname_change_events (address DESC);

CREATE TABLE slash_points (
                              node_address        TEXT NOT NULL,
                              slash_points        BIGINT NOT NULL,
                              reason              TEXT NOT NULL,
                              block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('slash_points');
CREATE INDEX ON slash_points (node_address DESC);

CREATE TABLE set_node_mimir (
                                address             TEXT NOT NULL,
                                key                 BIGINT NOT NULL,
                                value               TEXT NOT NULL,
                                block_timestamp     BIGINT NOT NULL
);

CALL setup_hypertable('set_node_mimir');

CREATE TABLE lp_detail
(
    pool                    TEXT   NOT NULL,
    asset_chain             TEXT,
    asset_addr              TEXT,
    asset_e8                BIGINT NOT NULL,
    stake_units             BIGINT NOT NULL,
    rune_addr               TEXT,
    rune_e8                 BIGINT NOT NULL,
    _asset_in_rune_e8  BIGINT      NOT NULL,
    pool_asset_depth        BIGINT NOT NULL,
    pool_rune_depth         BIGINT NOT NULL,
    pool_unit               BIGINT NOT NULL,
    pool_reward_per_unit    REAL   NOT NULL,
    pool_asset_fee_per_unit REAL   NOT NULL,
    pool_rune_fee_per_unit  REAL   NOT NULL,
    block_timestamp         BIGINT NOT NULL
);

CALL setup_hypertable('lp_detail');
