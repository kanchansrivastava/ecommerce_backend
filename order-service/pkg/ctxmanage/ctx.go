package ctxmanage

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"order-service/internal/auth"
	"order-service/middleware"
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

func GetAuthClaimsFromContext(ctx context.Context) (auth.Claims, error) {

	// checking if auth claims is present in the context or not
	// type assertion, making sure the value is of type auth.Claims
	claims, ok := ctx.Value(auth.ClaimsKey).(auth.Claims)
	if !ok {
		return auth.Claims{}, errors.New("claims not present in the context")
	}
	return claims, nil
}
