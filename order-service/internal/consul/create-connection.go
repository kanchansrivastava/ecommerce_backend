package consul

import (
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"time"
)

func CreateConnection() (*consulapi.Client, error) {
	// Create a default configuration object for the Consul client.
	config := consulapi.DefaultConfig()

	// Set the Consul server URL in the configuration.
	// This is where the Consul client will attempt to connect for service discovery or KV operations.
	config.Address = "http://consul.diwakar:8500"

	// Record the current time. This will be used to enforce a timeout of 10 minutes for the connection attempt.
	t := time.Now()

	// Declare variables to hold the potential Consul client and error for later use within the loop.
	var err error
	var client *consulapi.Client

	//Start an infinite loop, to continually attempt creating a connection to the Consul server until a client is successfully created.
	for {
		// Attempt to create a new Consul client with the given configuration.
		client, err = consulapi.NewClient(config)

		// Log the result of the Consul client creation for debugging purposes.
		fmt.Println("consul New Client status ", err)

		// If there was an error while creating the client:
		if err != nil {
			// Wait for 5 seconds before retrying to avoid overwhelming the server or network.
			time.Sleep(5 * time.Second)
			continue // Retry the connection attempt.
		}

		// If the Consul client was created successfully, check its connection status.
		var s string
		s, err = client.Status().Leader() // This fetches the current cluster leader's address.

		// If the attempt to retrieve the leader's status results in an error:
		if err != nil {
			// Log the error for debugging purposes.
			fmt.Println("consul connection status ", err)

			// Wait for 5 seconds before retrying.
			time.Sleep(5 * time.Second)
			continue // Retry the connection attempt.
		}

		// Print the leader's address to ensure the connection is established and the response is valid.
		fmt.Println(s)

		// Check if more than 10 minutes have passed since the beginning of the connection attempts.
		if time.Since(t) > 10*time.Minute {
			// If so, return an error indicating that the connection attempt timed out.
			return nil, fmt.Errorf("consul connection timeout")
		}

		// If successful (no errors and leader is reachable), break the loop to stop further retries.
		break
	}

	// After breaking the loop, double-check if there was any error during this process.
	// If there was (though unlikely after having broken out), return the error.
	if err != nil {
		return nil, err
	}

	// Return the successfully created Consul client to the caller.
	return client, nil
}
