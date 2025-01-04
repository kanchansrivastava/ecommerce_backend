package kafka

import (
	"context"
	"fmt"
	"github.com/twmb/franz-go/pkg/kgo"
	"log/slog"
	"os"
	"time"
)

type ConsumeResult struct {
	Record *kgo.Record
	Err    error
}

func ConsumeMessage(ctx context.Context, topicName string, ConsumerGroupName string, ch chan ConsumeResult) {

	host := os.Getenv("KAFKA_HOST")
	port := os.Getenv("KAFKA_PORT")
	if host == "" || port == "" {
		ch <- ConsumeResult{
			Err: fmt.Errorf("kafka host or port is empty"),
		}
		close(ch)
		return
	}

	seeds := []string{host + ":" + port}
	client, err := kgo.NewClient(
		// Seed brokers are the initial points of contact for the Kafka client.
		kgo.SeedBrokers(seeds...), // Provides broker addresses for the Kafka client.
		kgo.ConsumeTopics(topicName),
		kgo.ConsumerGroup(ConsumerGroupName),
		kgo.FetchMinBytes(1),
		kgo.FetchMaxWait(10*time.Millisecond),
	)

	if err != nil {
		ch <- ConsumeResult{
			Err: err,
		}
		close(ch)
		return
	}

	for {
		fetches := client.PollFetches(ctx)

		if errs := fetches.Errors(); len(errs) > 0 {
			// All errors are retried internally when fetching, but non-retriable errors are
			// returned from polls so that users can notice and take action.

			//maybe kafka is down
			slog.Error("ERROR: ", errs)
			time.Sleep(10 * time.Second)
			continue
		}

		// We can iterate through a record iterator...
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()

			ch <- ConsumeResult{
				Record: record,
				Err:    nil,
			}

		}

	}
}
