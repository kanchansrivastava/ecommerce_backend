package ctxmanage

import (
	"log/slog"
	"product-service/middleware"

	"github.com/gin-gonic/gin"
)

func GetTraceIdOfRequest(c *gin.Context) string {
	ctx := c.Request.Context()

	// ok is false if the type assertion was not successful
	traceId, ok := ctx.Value(middleware.TraceIdKey).(string)
	if !ok {
		slog.Error("trace id not present in the context")
		traceId = "Unknown"
	}
	return traceId
}
