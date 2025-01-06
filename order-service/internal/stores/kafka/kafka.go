package kafka

import (
	"context"
	"fmt"
	"github.com/twmb/franz-go/pkg/kgo"
	"log/slog"
	"os"
	"time"
)

type Conf struct {
	client   *kgo.Client
	consumer *kgo.Client
}

type ConsumeResult struct {
	Record *kgo.Record
	Err    error
}

// NewConf initializes a new Kafka configuration (`Conf`) object with the given topic and consumer group.
// This function sets up Kafka clients and handles connection retries with exponential backoff.
func NewConf(topic, ConsumerGroup string) (*Conf, error) {
	// Read Kafka connection details (host and port) from environment variables.
	host := os.Getenv("KAFKA_HOST")
	port := os.Getenv("KAFKA_PORT")

	// If either the host or port is not set, return an error.
	if host == "" || port == "" {
		return nil, fmt.Errorf("kafka host or port is empty")
	}

	// Combine the host and port into a connection string

	connString := fmt.Sprintf("%s:%s", host, port)

	var err error          // Placeholder for any errors encountered during setup.
	var client *kgo.Client // Placeholder for the Kafka producer client.

	// Retry loop to handle temporary failures when setting up the Kafka client.
	for i := 1; i < 8; i++ { // Try to connect up to 7 times.
		// Initialize a new Kafka client for producing messages.
		client, err = kgo.NewClient(
			kgo.SeedBrokers(connString),  // Configure Kafka endpoint using the broker's connection string.
			kgo.ProducerLinger(0),        // Messages won't linger; they are sent immediately.
			kgo.AllowAutoTopicCreation(), // Allow Kafka to automatically create the target topic if it doesn't exist.
		)

		// If the client creation fails, log the error and attempt a retry.
		if err != nil {
			fmt.Printf("kafka client error: %v\n", err)

			// Wait with exponential backoff before retrying.
			// Exponential backoff ensures we don't overload the broker with too frequent retries.
			time.Sleep(time.Duration(2*i) * time.Second)
			continue
		}

		// If the client was created successfully, test the connection by pinging the Kafka cluster.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*4) // Set a 4-second timeout for the ping.
		defer cancel()                                                          // Ensure context resources are cleaned up after the ping operation.
		err = client.Ping(ctx)

		// If the ping fails, log the error and attempt another retry.
		if err != nil {
			fmt.Printf("kafka client ping error: %v\n", err)
			time.Sleep(time.Duration(2*i) * time.Second)
			continue
		}

		// If no errors occur, the client setup is successful, and we break out of the retry loop.
		break
	}

	// If after all retries the client isn't set up, return an error indicating a failure to connect.
	if err != nil {
		return nil, fmt.Errorf("kafka client error: %v", err)
	}

	// Create another Kafka client specifically for the consumer.
	// This client is configured to consume messages from a given topic and consumer group.
	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(connString),           // Configure Kafka endpoint using the broker's connection string.
		kgo.ConsumeTopics(topic),              // Set the specific topic(s) this consumer will subscribe to.
		kgo.ConsumerGroup(ConsumerGroup),      // Assign the consumer to a specified consumer group.
		kgo.FetchMinBytes(1),                  // Minimum number of bytes to wait for in a fetch request.
		kgo.FetchMaxWait(10*time.Millisecond), // Maximum wait time before returning from a fetch request.
	)

	// Return the initialized configuration, which includes both producer and consumer clients.
	return &Conf{
		client:   client,   // Kafka producer client.
		consumer: consumer, // Kafka consumer client.
	}, nil
}

func (c *Conf) ProduceMessage(topicName string, key []byte, value []byte) error {

	// TODO:// figure out if we need context here or not
	ctx := context.Background()

	// Define a Kafka message (record) with the topic and the message value.
	record := &kgo.Record{Topic: topicName, Key: key, Value: value, Timestamp: time.Now().UTC()}

	// Configure Kafka log-level debugging to stderr.
	// 'kgo.WithLogger' is used to enable Kafka client logging for debugging purposes.
	kgo.WithLogger(kgo.BasicLogger(os.Stdout, kgo.LogLevelDebug, nil))

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

func (c *Conf) ConsumeMessage(ctx context.Context, topicName string, ConsumerGroupName string, ch chan ConsumeResult) {

	host := os.Getenv("KAFKA_HOST")
	port := os.Getenv("KAFKA_PORT")
	if host == "" || port == "" {
		ch <- ConsumeResult{
			Err: fmt.Errorf("kafka host or port is empty"),
		}
	}

	for {
		fetches := c.consumer.PollFetches(ctx)

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
