#!/bin/bash

# This script completely purges all aggregates and resets the consumer to the start

BOOTSTRAP_SERVER=localhost:9094
BLOCK_EVENTS_TOPIC=block-events
EVENT_GROUP=pool-emitter
EVENT_TOPIC=pool-events
STATS_TOPIC=pool-stats
STATS_TABLE=pool-stats-table
AGG_GROUP=pool-agg-stats
AGG_TABLE=pool-agg-stats-table

kafka-consumer-groups \
    --bootstrap-server ${BOOTSTRAP_SERVER} \
    --group ${EVENT_GROUP} \
    --topic ${BLOCK_EVENTS_TOPIC} \
    --reset-offsets \
    --to-offset 0 \
    --execute

kafka-consumer-groups \
    --bootstrap-server ${BOOTSTRAP_SERVER} \
    --group ${STATS_TOPIC} \
    --topic ${EVENT_TOPIC} \
    --reset-offsets \
    --to-offset 0 \
    --execute

kafka-consumer-groups \
    --bootstrap-server ${BOOTSTRAP_SERVER} \
    --group ${AGG_GROUP} \
    --topic ${STATS_TOPIC} \
    --reset-offsets \
    --to-offset 0 \
    --execute

kafka-topics --delete --bootstrap-server ${BOOTSTRAP_SERVER} --topic ${EVENT_TOPIC} --if-exists
kafka-topics --delete --bootstrap-server ${BOOTSTRAP_SERVER} --topic ${STATS_TOPIC} --if-exists
kafka-topics --delete --bootstrap-server ${BOOTSTRAP_SERVER} --topic ${STATS_TABLE} --if-exists
kafka-topics --delete --bootstrap-server ${BOOTSTRAP_SERVER} --topic ${AGG_TABLE} --if-exists

kafka-topics --create --bootstrap-server localhost:9094 --topic ${EVENT_TOPIC} --partitions 10 \
    --config segment.bytes=107374182 --config retention.bytes=8368709120

kafka-topics --create --bootstrap-server localhost:9094 --topic ${STATS_TOPIC} --partitions 3 \
    --config segment.bytes=57374182 --config retention.bytes=436870912

kafka-topics --create --bootstrap-server localhost:9094 --topic ${STATS_TABLE} --partitions 10 \
    --config segment.bytes=10737418 --config retention.bytes=4368709  --config cleanup.policy=compact

kafka-topics --create --bootstrap-server localhost:9094 --topic ${AGG_TABLE} --partitions 3 \
    --config segment.bytes=10737418 --config retention.bytes=4368709  --config cleanup.policy=compact
