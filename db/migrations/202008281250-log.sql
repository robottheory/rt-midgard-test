-- +migrate Up

CREATE TABLE block_log (
	height			BIGINT NOT NULL,
	timestamp		BIGINT NOT NULL,
	hash			BYTEA NOT NULL,
	agg_state		BYTEA,
	PRIMARY KEY (height)
);


-- +migrate Down

DROP TABLE block_log;
