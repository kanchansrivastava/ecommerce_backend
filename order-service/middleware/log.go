package middleware

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
	"time"
)

type key string

const TraceIdKey key = "1"

// -> logger - > check -> models - > service
func Logger() gin.HandlerFunc {

	return func(c *gin.Context) {
		requestStartTime := time.Now()

		// get a trace id
		traceId := uuid.NewString()

		//fetching the context container from the request.context()
		ctx := c.Request.Context()
		//adding the value in the context
		ctx = context.WithValue(ctx, TraceIdKey, traceId)
		// The 'WithContext' method on 'c.Request' creates a new copy of the request ('req'),
		// but with an updated context ('ctx') that contains our trace ID.
		// The original request does not get changed by this; we're simply creating a new version of it ('req').
		c.Request = c.Request.WithContext(ctx)
		// Now, we want to carry forward this updated request (that has the new context) through our application.
		// So, we replace 'c.Request' (the original request) with 'req' (the new version with the updated context).
		// After this line, when we use 'c.Request' in this function or pass it to others, it'll be this new version
		// that carries our trace ID in its context.

		slog.Info("started", slog.String("TRACE ID", traceId),
			slog.String("Method", c.Request.Method), slog.Any("URL Path", c.Request.URL.Path))

		//we use c.Next only when we are using r.Use() method to assign middlewares
		c.Next() // call next thing in the chain

		slog.Info("completed", slog.String("TRACE ID", traceId),
			slog.String("Method", c.Request.Method), slog.Any("URL Path", c.Request.URL.Path),
			slog.Int("Status Code", c.Writer.Status()), slog.Int64("duration μs,",
				time.Since(requestStartTime).Microseconds()))
	}
}
