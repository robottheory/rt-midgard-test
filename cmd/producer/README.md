The `producer` command reads blocks from a thornode, extracts the events, and sends them to a kafka topic.  Each event is a message and the key is the block plus the offset of the event within the block.

When it starts, it finds the last offset it sent to the topic and resumes from there.
