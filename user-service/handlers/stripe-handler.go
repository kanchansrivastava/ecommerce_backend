package handlers

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"user-service/pkg/ctxmanage"
	"user-service/pkg/logkey"
)

func (h *Handler) GetStripeDetails(c *gin.Context) {
	// Get the traceId from the request for tracking logs
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	claims, err := ctxmanage.GetAuthClaimsFromContext(c.Request.Context())
	if err != nil {
		slog.Error(
			"missing claims",
			slog.String(logkey.TraceID, traceId), slog.Any(logkey.ERROR, err.Error()))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "User Id is required"})
		return
	}

	userId := claims.Subject
	ctx := c.Request.Context()
	stripeCustomerId, err := h.u.GetStripeCustomerID(ctx, userId)
	if err != nil {
		slog.Error(
			"failed to get stripe customer id",
			slog.String(logkey.TraceID, traceId), slog.Any(logkey.ERROR, err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to get stripe customer id"})
		return
	}
	slog.Info("successfully got stripe customer id", slog.String(logkey.TraceID, traceId))
	c.JSON(http.StatusOK, gin.H{"stripe_customer_id": stripeCustomerId})

}
