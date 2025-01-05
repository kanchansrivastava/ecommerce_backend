package handlers

import (
	"cart-service/pkg/ctxmanage"
	"cart-service/pkg/logkey"
	"encoding/json"
	"fmt"
	"log/slog"

	"cart-service/internal/auth"
	"cart-service/internal/consul"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProductServiceResponse struct {
	ProductID string `json:"product_id"`
	Stock     int    `json:"stock"`
	PriceID   string `json:"price_id"`
}

func (h *Handler) AddToCart(c *gin.Context) {
	// Get the traceId for logging
	traceId := ctxmanage.GetTraceIdOfRequest(c)
	claims, ok := c.Request.Context().Value(auth.ClaimsKey).(auth.Claims)
	if !ok {
		slog.Error("claims not found", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId := claims.Subject

	// Parse the request body
	var request struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		slog.Error("invalid request body", slog.String(logkey.TraceID, traceId), slog.String(logkey.ERROR, err.Error()))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	// Validate the request data
	if request.ProductID == "" || request.Quantity <= 0 {
		slog.Error("invalid product ID or quantity", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Product ID and quantity must be valid"})
		return
	}

	// Discover product service using Consul
	address, port, err := consul.GetServiceAddress(h.client, "products")
	if err != nil {
		slog.Error("product service unavailable", slog.String(logkey.TraceID, traceId), slog.String(logkey.ERROR, err.Error()))
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "Product service is unavailable"})
		return
	}

	// Fetch product details from the product service
	httpQuery := fmt.Sprintf("http://%s:%d/products/stock/%s", address, port, request.ProductID)
	resp, err := http.Get(httpQuery)
	if err != nil || resp.StatusCode != http.StatusOK {
		slog.Error("error fetching product details", slog.String(logkey.TraceID, traceId), slog.Any(logkey.ERROR, err))
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"message": "Failed to fetch product details"})
		return
	}
	defer resp.Body.Close()

	// Decode the response
	var stockPriceData ProductServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&stockPriceData); err != nil {
		slog.Error("error decoding product details", slog.String(logkey.TraceID, traceId), slog.String(logkey.ERROR, err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to process product details"})
		return
	}

	// Check stock availability
	if request.Quantity > stockPriceData.Stock {
		slog.Error("insufficient stock", slog.String(logkey.TraceID, traceId),
			slog.String("ProductID", request.ProductID), slog.Int("Requested", request.Quantity), slog.Int("Available", stockPriceData.Stock))
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"message": "Insufficient stock available"})
		return
	}

	// Add to cart
	err = h.cConf.AddToCartDB(c.Request.Context(), userId, request.ProductID, request.Quantity, stockPriceData.Stock, stockPriceData.PriceID)
	if err != nil {
		slog.Error("error adding product to cart", slog.String(logkey.TraceID, traceId),
			slog.String(logkey.ERROR, err.Error()), slog.String("ProductID", request.ProductID), slog.Int("Quantity", request.Quantity))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to add product to cart"})
		return
	}

	// Log success
	slog.Info("product added to cart", slog.String("Trace ID", traceId),
		slog.String("ProductID", request.ProductID), slog.Int("Quantity", request.Quantity), slog.String("UserID", userId))

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "Product added to cart successfully"})
}

func (h *Handler) GetActiveCartItems(c *gin.Context) {
	// Get the traceId for logging
	traceId := ctxmanage.GetTraceIdOfRequest(c)
	claims, ok := c.Request.Context().Value(auth.ClaimsKey).(auth.Claims)
	if !ok {
		slog.Error("claims not found", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId := claims.Subject

	// Fetch active cart items for the user
	cartResponse, err := h.cConf.GetActiveCartItems(c.Request.Context(), userId)
	if err != nil {
		slog.Error("error fetching active cart items", slog.String(logkey.TraceID, traceId), slog.String(logkey.ERROR, err.Error()), slog.String("UserID", userId))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch cart items"})
		return
	}

	// Log success
	slog.Info("fetched active cart items", slog.String("Trace ID", traceId), slog.String("UserID", userId))

	// Respond with the cart details
	c.JSON(http.StatusOK, gin.H{
		"items": cartResponse.Items,
	})
}
