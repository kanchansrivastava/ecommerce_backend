package consul

import (
	"errors"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"os"
	"strconv"
)

func RegisterWithConsul() (*consulapi.Client, string, error) {
	// Step 1: Read configuration from environment variables
	// By default, in Docker, the value of HOSTNAME is set to the
	// Docker container's IP address, but could also be hostname.
	hostName := os.Getenv("HOSTNAME")                         // Get the hostname (e.g., container's hostname or machine name)
	svcName := os.Getenv("SERVICE_NAME")                      // Get the service name
	portString := os.Getenv("APP_PORT")                       // Get the application port
	consulAddress := os.Getenv("CONSUL_HTTP_ADDRESS")         // Get the address of the Consul server
	svcEndpointPrefix := os.Getenv("SERVICE_ENDPOINT_PREFIX") // Service endpoint prefix for KV store

	// Step 1.1: Validate that all required environment variables are set
	if hostName == "" || svcName == "" || portString == "" ||
		consulAddress == "" || svcEndpointPrefix == "" {
		// Return an error if any required environment variable is missing.
		return nil, "", errors.New(
			`env variables not set for hostName, 
                 svcName, port, consulAddress, svcEndpointPrefix`)
	}

	// Step 2: Convert the port string to an integer
	// This is important because the service expects the port to be an int, not a string
	port, err := strconv.Atoi(portString) // Convert APP_PORT from string to int
	if err != nil {
		return nil, "", fmt.Errorf("port is not a number: %w", err) // Handle invalid port values
	}

	// Step 3: Create a Consul client configuration
	// The Consul client provides default configuration settings through `DefaultConfig`.
	config := consulapi.DefaultConfig()

	// Step 3.1: Set the Consul server address
	// Address is the location of the Consul server we want to register with.
	config.Address = consulAddress

	// Step 4: Initialize a new Consul client
	// The `NewClient` function establishes the connection to the Consul agent/server.
	client, err := consulapi.NewClient(config)
	if err != nil {
		// If there is a problem creating the client, return an error.
		return nil, "", fmt.Errorf("create consul client: %w", err)
	}

	// Step 5: Prepare service registration information
	// Create a new instance of `AgentServiceRegistration` to describe our service.
	registration := consulapi.AgentServiceRegistration{}

	// Step 5.1: Generate a unique registration ID
	// This ensures each registered service instance is unique, even if the service name is the same.
	regId := svcName + "-" + hostName
	registration.ID = regId // Use hostname to create a unique service ID

	// Step 5.2: Set the name of the service
	// This name is how other clients will discover or query the service via Consul.
	registration.Name = svcName

	// Step 5.3: Set the host and port where the service runs
	// Address is the machine's hostname or IP, and Port is the app's listening port.
	registration.Address = hostName // IP/hostname where the service is running
	registration.Port = port        // Port at which the service can be accessed

	// Step 6: Define health check options
	// A health check ensures that the service is running when queried by Consul.
	registration.Check = &consulapi.AgentServiceCheck{
		// Step 6.1: HTTP health check endpoint
		// This is a URL that Consul queries periodically to check if the service is healthy.
		HTTP: fmt.Sprintf("http://%s:%d/ping", hostName, port),

		// Step 6.2: Health check interval
		// Consul executes health checks periodically. Here, every 30 seconds.
		Interval: "30s",

		// Step 6.3: Timeout for health check responses
		// If the service does not respond within 10 seconds, it's considered unhealthy.
		Timeout: "10s",

		// Step 6.4: Deregister a service if it remains unhealthy for too long
		// If Consul marks the service as critical for more than 30 seconds,
		// it will automatically deregister the service.
		DeregisterCriticalServiceAfter: "30s",
	}

	// Step 7: Register the service with Consul
	// Log the registration process
	fmt.Println("registering service with consul")

	// Register the service registration details with Consul
	err = client.Agent().ServiceRegister(&registration)
	if err != nil {
		// If the registration fails, return an error with details.
		return nil, "", fmt.Errorf("register service with consul: %w", err)
	}

	// Step 8: Add service information to Consul's Key-Value (KV) store
	// Define the key-value pair to store service metadata in Consul's KV store
	kv := client.KV()         // Get the KV store client
	pair := consulapi.KVPair{ // Create a key-value entry
		Key:   svcEndpointPrefix, // Unique key for the service endpoint prefix
		Value: []byte(svcName),   // Service name stored as the value (in bytes)
	}

	// Add the key-value pair to the Consul KV store
	_, err = kv.Put(&pair, nil)
	if err != nil {
		// If storing the key-value pair fails, return an error.
		return nil, "", fmt.Errorf("register key in consul KV store: %w", err)
	}

	// Step 9: Return the Consul client and registration ID
	// Everything worked correctly, so return:
	// - the Consul client
	// - the registration ID
	// - no error
	return client, regId, nil
}
