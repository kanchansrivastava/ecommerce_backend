package kafka

import (
	"context"
	"github.com/twmb/franz-go/pkg/kgo"
	"log/slog"
	"time"
)

type ConsumeResult struct {
	Record *kgo.Record
}

// ConsumeMessage is responsible for consuming messages from Kafka
// and sending them to the provided channel `ch`.
// This function continuously polls for messages using a Kafka consumer
func (c *Conf) ConsumeMessage(ctx context.Context, ch chan ConsumeResult) {

	// Infinite loop to continuously poll messages from Kafka
	for {
		// Poll messages from Kafka using the Kafka consumer available in `c.consumer`.
		// `PollFetches` fetches the messages from Kafka topics
		fetches := c.consumer.PollFetches(ctx)

		// Check if there are any errors in the fetch result
		// This might occur if Kafka is unavailable or there are connection issues.
		if errs := fetches.Errors(); len(errs) > 0 {
			// Log the errors to help diagnose what went wrong (e.g., Kafka being down).
			slog.Error("ERROR: ", errs)

			// If there's an error (e.g., temporary network issues or Kafka being down),
			// wait for 5 seconds before retrying to avoid overwhelming the system.
			time.Sleep(5 * time.Second)
			continue
		}

		// Get an iterator for the records fetched from Kafka
		iter := fetches.RecordIter()

		// Loop through the iterator until all records are processed
		for !iter.Done() {
			// Get the next Kafka record from the fetch result
			rec := iter.Next()

			// Send the fetched record to the provided channel `ch` for processing
			ch <- ConsumeResult{
				Record: rec,
			}
		}

		// The loop will continue, polling fetches and processing records indefinitely
	}
}
