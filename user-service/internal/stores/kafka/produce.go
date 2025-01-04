package kafka

import (
	"context"
	"fmt"
	"github.com/twmb/franz-go/pkg/kgo"
	"time"
)

func (c *Conf) ProduceMessage(topicName string, key []byte, value []byte) error {

	ctx := context.Background()

	// Define a Kafka message (record) with the topic and the message value.
	record := &kgo.Record{Topic: topicName, Key: key, Value: value, Timestamp: time.Now().UTC()}

	/* ------------------ Producing a Kafka Message ------------------ */
	// Produce the record (message) synchronously.
	// Use 'ProduceSync' to send a message and wait for the broker’s response.
	// This method is synchronous, so it waits for acknowledgment from Kafka.
	pr := c.client.ProduceSync(ctx, record)

	// Check if the production was successful by inspecting the response.
	// 'pr.FirstErr()' gets the first record’s response and check for any error while producing.
	//rec, err := pr.First()
	err := pr.FirstErr()
	if err != nil {
		// If there’s an error while producing the message, log it.
		fmt.Printf("Record had a produce error while synchronously producing: %v\n", err)
		return err
	}

	return nil
	//// Print a success message with details about the produced record.
	//fmt.Printf("Record produced successfully! Topic: %s, Partition: %d, Offset: %d, Timestamp: %s, Key: %v, Value: %s\n",
	//	rec.Topic,                          // The topic the message was sent to.
	//	rec.Partition,                      // The partition the message was stored in.
	//	rec.Offset,                         // The offset of the message within the partition.
	//	rec.Timestamp.Format(time.RFC3339), // The timestamp of the message.
	//	rec.Key,                            // The key of the message (can be nil if no key is provided).
	//	rec.Value)                          // The actual message data.
}
