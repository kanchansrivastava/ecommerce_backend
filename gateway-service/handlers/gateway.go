package handlers

import (
	"fmt"
	"gateway-service/internal/consul"
	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
	"io"
	"net/http"
	"strings"
)

type Handler struct {
	client *consulapi.Client
}

func NewHandler(client *consulapi.Client) *Handler {
	return &Handler{
		client: client,
	}
}

func (h *Handler) APIGateway(c *gin.Context) {
	// Retrieve the placeholder value for "path" from the request's URL.
	// For example, if the request is `/api/users/create/123`, the `path` would be `users/create/123`.
	fullPath := c.Param("path") // give full path /users/create/123
	// or c.Request.URL.Path // fetch the endpoint
	// Split the URL path into segments using "/" as the delimiter.

	segments := strings.Split(fullPath, "/")

	var serviceEndpoint string

	// Check if there are URL segments after splitting and if the first segment is not empty.
	// If it isn't valid, simply return without doing anything.
	if len(segments) > 1 && segments[1] != "" {
		serviceEndpoint = segments[1]
	} else {
		return // Exit if no valid service endpoint exists.
	}

	// Print the service endpoint to understand which service is being targeted.
	fmt.Println(serviceEndpoint)

	// Use the service endpoint to query the Key-Value (KV) storage in Consul to fetch service metadata.
	pair, _, err := h.client.KV().Get(serviceEndpoint, nil)

	// Check if the call to the KV store failed or the `pair` returned is `nil`.
	// If either happens, it means the requested service could not be found.
	if err != nil || pair == nil {
		// Abort the current request with an HTTP 404 status and return an error message.
		c.AbortWithStatusJSON(
			http.StatusNotFound,
			gin.H{"error": "Service not found"},
		)
		// Log that the service could not be found.
		fmt.Println("Service not found for " + c.Request.URL.Path)
		return
	}

	// Extract the service name from the value stored in the KV store.
	serviceName := string(pair.Value)

	// Print the service name.
	fmt.Println("Service name is " + serviceName)

	// Use a helper function `consul.GetService` to fetch the service's address and port.
	// This ensures the service is available and provides the information to redirect the request.
	serviceAddress, servicePort, err := consul.GetService(h.client, serviceName)
	if err != nil {
		// If the service is not accessible, abort the request and return an HTTP 503 error.
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to reach service"})
		// Log the error for debugging.
		fmt.Println(err)
		return
	}

	// Get the request's context to pass it downstream.
	ctx := c.Request.Context()

	// Construct the complete HTTP query to reach the backend service dynamically.
	// This includes the service address, port, and the original path the client requested.
	httpQuery := fmt.Sprintf("http://%s:%d%s", serviceAddress, servicePort, fullPath)
	fmt.Println(httpQuery)

	// Create a new HTTP request with the same method and body as the original client request.
	req, err := http.NewRequestWithContext(ctx, c.Request.Method, httpQuery, c.Request.Body)
	if err != nil {
		// If the request creation fails, return an HTTP 500 error and exit.
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copy all the headers from the original client request into the new request.
	req.Header = c.Request.Header

	// Forward the prepared request to the backend service using the default HTTP client.
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		// If the HTTP call to the backend service fails, return an HTTP 500 error and exit.
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to reach service"})
		return
	}
	// Ensure the response body is properly closed after reading it.
	defer resp.Body.Close()

	// Read the response body from the backend service.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// If there is an error while reading the response, return an HTTP 500 error and exit.
		fmt.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Forward the backend service's response directly to the client.
	// The status code and content type are also propagated to ensure consistency.
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}
