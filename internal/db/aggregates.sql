
-- version 1

DROP SCHEMA IF EXISTS midgard_agg CASCADE;
CREATE SCHEMA midgard_agg;

-- TODO(huginn): decide if we want to move this view into it's usage place (members.go)

CREATE VIEW midgard_agg.pending_adds AS
SELECT *
FROM pending_liquidity_events AS p
WHERE pending_type = 'add'
  AND NOT EXISTS(
    -- Filter out pending liquidity which was already added
        SELECT *
        FROM stake_events AS s
        WHERE
                p.rune_addr = s.rune_addr
          AND p.pool=s.pool
          AND p.block_timestamp <= s.block_timestamp)
  AND NOT EXISTS(
    -- Filter out pending liquidity which was withdrawn without adding
        SELECT *
        FROM pending_liquidity_events AS pw
        WHERE
                pw.pending_type = 'withdraw'
          AND p.rune_addr = pw.rune_addr
          AND p.pool = pw.pool
          AND p.block_timestamp <= pw.block_timestamp);

CREATE TABLE midgard_agg.watermarks (
                                        materialized_table VARCHAR(60) PRIMARY KEY,
                                        watermark BIGINT NOT NULL
);

CREATE FUNCTION midgard_agg.watermark(t VARCHAR) RETURNS BIGINT
    LANGUAGE SQL STABLE AS $$
SELECT watermark FROM midgard_agg.watermarks
WHERE materialized_table = t;
$$;

CREATE PROCEDURE midgard_agg.refresh_watermarked_view(t VARCHAR, w_new BIGINT)
    LANGUAGE plpgsql AS $BODY$
DECLARE
w_old BIGINT;
BEGIN
SELECT watermark FROM midgard_agg.watermarks WHERE materialized_table = t
    FOR UPDATE INTO w_old;
EXECUTE format($$
                   INSERT INTO midgard_agg.%1$I_materialized
		SELECT * from midgard_agg.%1$I
			WHERE $1 <= block_timestamp AND block_timestamp < $2
	$$, t) USING w_old, w_new;
UPDATE midgard_agg.watermarks SET watermark = w_new WHERE materialized_table = t;
END
$BODY$;