package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"user-service/internal/stores/kafka"
	"user-service/internal/users"
	"user-service/pkg/ctxmanage"
	"user-service/pkg/logkey"
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

/*
	when a user logs in, create a token for the user if login is a success
	and return the token back to the client
*/

func (h *Handler) Login(c *gin.Context) {

	// Get the trace ID from the request for debugging/tracking
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	// Declare a struct to hold the login request payload
	var loginPayload struct {
		Email    string `json:"email" validate:"required,email"` // Email must be valid and required
		Password string `json:"password" validate:"required"`    // Password required
	}

	// Bind the JSON request body into loginPayload struct
	err := c.ShouldBindJSON(&loginPayload)

	// Check if JSON bind resulted in errors
	if err != nil {
		// Log the JSON bind error
		slog.Error("JSON validation error", slog.String(logkey.TraceID, traceId),
			slog.String(logkey.ERROR, err.Error()))

		// Respond with a Bad Request error
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid email or password"})
		return
	}

	// Validate the loginPayload fields
	err = h.validate.Struct(loginPayload)

	// Check if validation failed
	if err != nil {

		// Log generic validation failure
		slog.Error("Validation failed", slog.String(logkey.TraceID, traceId),
			slog.String(logkey.ERROR, err.Error()))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid email or password"})
		return
	}

	// Proceed to authenticate the user by verifying credentials
	userData, claims, err := h.u.Authenticate(c.Request.Context(), loginPayload.Email, loginPayload.Password)

	// Check if authentication failed
	if err != nil {
		slog.Error("Authentication failed", slog.String(logkey.TraceID, traceId), slog.String(logkey.ERROR, err.Error()))

		// For incorrect credentials, send an unauthorized error
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, users.ErrInvalidPassword) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		// If another error occurred, respond with an internal server error
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
		return
	}

	token, err := h.a.GenerateToken(claims)
	if err != nil {
		slog.Error("Error in generating token", slog.String(logkey.TraceID, traceId), slog.String(logkey.ERROR, err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
		return
	}

	// If login is successful, return the user data in the response
	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    userData,
		"token":   token,
	})
}
