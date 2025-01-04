package consul

import (
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"math/rand"
	"time"
)

func GetService(client *consulapi.Client, serviceName string) (string, int, error) {
	// Use the Consul API `Health().Service()` method to get information about the services
	// - `serviceName`: The name of the service to find.
	// - The empty string `""`: Specifies no tag filter is applied.
	// - `true`: Only return services that are "healthy" (have passing health checks).
	// - `nil`: No context or additional query options are provided.
	services, _, err := client.Health().Service(serviceName, "", true, nil)

	// If there is an error in querying Consul or if no healthy services are found, return:
	// - An empty string for the address.
	// - Port `0` as the service port.
	// - The error (if any) for further handling upstream.
	if err != nil || len(services) == 0 {
		return "", 0, err
	}

	// Declare a variable to store the selected service (later picked from the list of returned services).
	var service *consulapi.ServiceEntry

	// Debug: Print all available services retrieved from Consul for logging or debugging purposes.
	fmt.Println(services)

	// If more than one healthy instance of the requested service exists:
	if len(services) > 1 {
		fmt.Println("more than one service")
		fmt.Printf("%+v\n", services) // Log the details of all available services.

		// Create a new random number generator (RNG) instance with a seed based on the current time.
		// This ensures a different random service is selected each time.
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)

		// Generate a random index in the range `[0, len(services)-1]` to randomly pick one of the available services.
		randomServiceIndex := rng.Intn(len(services)) // For example, 3 services => index range: 0-2

		// Select the service at the randomly generated index.
		service = services[randomServiceIndex]
	} else {
		// If there is only one healthy service instance, simply select it.
		service = services[0]
	}

	// Return the selected service's properties:
	// - Service's address as a string (e.g., IP or hostname).
	// - Service's port as an integer.
	// - `nil` to indicate no errors occurred.
	return service.Service.Address, service.Service.Port, nil
}
