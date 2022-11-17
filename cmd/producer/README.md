The `producer` command reads blocks from a thornode, extracts the events, and sends them to a kafka topic.  Each event is a message and the key is the block plus the offset of the event within the block.

When it starts, it finds the last offset it sent to the topic and resumes from there.

The topic to receive events must exist before the command runs.  See below for an example of how to create a topic.  In this case, it is sized to retain all the events forever.

```bash
kafka-topics --create \
             --bootstrap-server localhost:9094 \
             --topic block-events \
             --partitions 1 \
             --config retention.bytes=83687091200 \
             --config retention.ms=-1
```

Reset state:
1) Delete topic: `kafka-topics --delete --bootstrap-server localhost:9094 --topic block-events`
2) Create topic: `kafka-topics --create --bootstrap-server localhost:9094 --topic block-events --partitions 1 --config retention.bytes=643687091200 --config retention.ms=-1`


