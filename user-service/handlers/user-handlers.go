package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"user-service/internal/stores/kafka"
	"user-service/internal/users"
	"user-service/pkg/ctxmanage"
	"user-service/pkg/logkey"

	"github.com/gin-gonic/gin"
)

// Signup handles the user signup process.
// It validates the incoming JSON request, ensures it doesn't exceed a size limit,
// creates a new user in the database, and sends a Kafka message indicating the account was created.
func (h *Handler) Signup(c *gin.Context) {
	// Retrieve the trace ID for the current request for logging and tracing purposes.
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Check if the size of the request body exceeds the 5KB limit (1 KB = 1024 Bytes).
	if c.Request.ContentLength > 5*1024 {
		// Log an error indicating that the request body size limit was breached.
		slog.Error("request body limit breached",
			slog.String(logkey.TraceID, traceId),
			slog.Int64("Size Received", c.Request.ContentLength),
		)

		// Respond with HTTP 400 Bad Request and an appropriate error message.
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "payload exceeding size limit",
		})
		return
	}

	// Parse the incoming JSON request into a `NewUser` struct.
	var newUser users.NewUser
	err := c.ShouldBindJSON(&newUser)
	if err != nil {
		// Log an error if JSON parsing or validation fails, along with the trace ID.
		slog.Error("json validation error",
			slog.String(logkey.TraceID, traceId),
			slog.String(logkey.ERROR, err.Error()),
		)

		// Respond with HTTP 400 Bad Request and indicate the error.
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": http.StatusText(http.StatusBadRequest),
		})
		return
	}

	// Validate the parsed `NewUser` struct using `h.validate` (likely a validator library instance).
	err = h.validate.Struct(newUser)
	if err != nil {
		// Log an error if validation fails, along with the trace ID and specific error message.
		slog.Error("validation failed",
			slog.String(logkey.TraceID, traceId),
			slog.String(logkey.ERROR, err.Error()),
		)

		// Respond with HTTP 400 Bad Request and indicate the need for correct input formats.
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "please provide values in correct format",
		})
		return
	}

	// Extract the context from the HTTP request to pass it to the service layer.
	ctx := c.Request.Context()

	// Attempt to insert the new user into the database using the `InsertUser` method.
	user, err := h.u.InsertUser(ctx, newUser)
	if err != nil {
		// Log an error if user creation fails, along with the trace ID and specific error message.
		slog.Error("error in creating the user",
			slog.String(logkey.TraceID, traceId),
			slog.String(logkey.ERROR, err.Error()),
		)

		// Respond with HTTP 500 Internal Server Error indicating that user creation failed.
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "User Creation Failed",
		})
		return
	}

	// Send a Kafka message asynchronously in a separate goroutine after user creation.
	go func() {
		// Marshal the created user's data into JSON format for the Kafka message payload.
		data, err := json.Marshal(user)
		if err != nil {
			// Log an error if JSON marshaling fails, along with the trace ID.
			slog.Error("error in marshaling user",
				slog.String(logkey.TraceID, traceId),
				slog.String(logkey.ERROR, err.Error()),
			)
			return
		}

		// Use the user's ID as the Kafka message key.
		key := []byte(user.ID)

		// Attempt to send the Kafka message using the `ProduceMessage` method.
		err = h.k.ProduceMessage(kafka.TopicAccountCreated, key, data)
		if err != nil {
			// Log an error if producing the Kafka message fails, along with the trace ID.
			slog.Error("error in producing message",
				slog.String(logkey.TraceID, traceId),
				slog.String(logkey.ERROR, err.Error()),
			)
			return
		}
	}()

	// Respond with HTTP 200 OK and return the created user's data as JSON.
	c.JSON(http.StatusOK, user)
}

var credentials struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) UserLogin(c *gin.Context) {
	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := c.Request.Context()

	user, err := h.u.FetchUser(ctx, credentials.Email, credentials.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user": gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	}})
}
