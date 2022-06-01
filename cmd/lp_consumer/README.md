The `lp_consumer` command consumes block events create by the `producer` command and 
calculates various per-pool and aggregate statistics.

Reset the consumer:
```bash
kafka-consumer-groups \
    --bootstrap-server localhost:9092 \
    --group my-consumer-group \
    --topic myTopicName \
    --reset-offsets \
    --to-offset 0 \
    --execute
```
You can add this function:
```go
func (e myConsumerGroup) Setup(sess sarama.ConsumerGroupSession) error {
    sess.ResetOffset(topic, partition, offset, "")

    return nil
}
```
