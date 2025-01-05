package kafka

import (
	"context"
	"fmt"
	"github.com/twmb/franz-go/pkg/kgo"
	"os"
	"time"
)

type Conf struct {
	client   *kgo.Client
	consumer *kgo.Client
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
