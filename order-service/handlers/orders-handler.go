package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"order-service/internal/auth"
	"order-service/internal/consul"
	"order-service/internal/orders"
	"order-service/pkg/ctxmanage"
	"order-service/pkg/logkey"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

type CartItem struct {
	ProductID string `json:"product_id"` // Unique identifier for the product
	Quantity  int    `json:"quantity"`   // Quantity of the product in the cart
}

type CartResponse struct {
	Items []CartItem `json:"items"` // List of items in the cart
}

type ProductServiceResponse struct {
	ProductID string `json:"product_id"` // Unique identifier for the product
	Name      string `json:"name"`       // Name of the product
	Price     int    `json:"price"`      // Price per unit of the product (in smallest currency unit)
	Stock     int    `json:"stock"`      // Available stock of the product
	PriceID   string `json:"price_id"`
}

type UserServiceResponse struct {
	StripCustomerId string `json:"stripe_customer_id"`
}

func (h *Handler) Checkout(c *gin.Context) {
	traceId := ctxmanage.GetTraceIdOfRequest(c)
	claims, ok := c.Request.Context().Value(auth.ClaimsKey).(auth.Claims)
	if !ok {
		slog.Error("claims not found", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": http.StatusUnauthorized})
		return
	}

	// Validate Consul client initialization
	if h.client == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "consul client is not initialized"})
		return
	}

	// Fetch Stripe customer ID
	userServiceResponse, err := h.fetchUserStripeCustomerID(c.Request.Context(), c.Request.Header.Get("Authorization"), traceId)
	if err != nil || userServiceResponse.StripCustomerId == "" {
		slog.Error("failed to fetch Stripe customer ID", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Stripe customer ID"})
		return
	}

	// Fetch detailed cart items
	cartItems, err := h.fetchCartItems(c.Request.Context(), c.Request.Header.Get("Authorization"), traceId)
	if err != nil {
		slog.Error("failed to fetch cart items", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart items"})
		return
	}

	detailedItems, err := h.fetchProductDetails(c.Request.Context(), cartItems, c.Request.Header.Get("Authorization"), traceId)
	if err != nil || len(detailedItems) == 0 {
		slog.Error("failed to fetch product details", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product details"})
		return
	}

	// Stripe configuration
	sKey := os.Getenv("STRIPE_TEST_KEY")
	if sKey == "" {
		slog.Error("Stripe secret key not found", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Stripe secret key not found"})
		return
	}
	stripe.Key = sKey

	// Prepare Stripe line items
	orderId := uuid.NewString()
	lineItems := []*stripe.CheckoutSessionLineItemParams{}
	var jsonLineItems []map[string]interface{}

	for _, item := range detailedItems {
		if item.Stock < item.Quantity || item.PriceID == "" {
			slog.Error("invalid product details", slog.String("product_id", item.ProductID), slog.String(logkey.TraceID, traceId))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid product in cart"})
			return
		}
		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Price:    stripe.String(item.PriceID),
			Quantity: stripe.Int64(int64(item.Quantity)),
		})
		jsonLineItems = append(jsonLineItems, map[string]interface{}{
			"product_id": item.ProductID,
			"quantity":   item.Quantity,
		})
	}

	jsonOutput, err := json.Marshal(jsonLineItems)
	if err != nil {
		slog.Error("failed to marshal line items", slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare line items"})
		return
	}

	// Create Stripe checkout session
	params := &stripe.CheckoutSessionParams{
		Customer:                 stripe.String(userServiceResponse.StripCustomerId),
		SubmitType:               stripe.String("pay"),
		Currency:                 stripe.String(string(stripe.CurrencyINR)),
		BillingAddressCollection: stripe.String("auto"),
		LineItems:                lineItems,
		Mode:                     stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:               stripe.String("https://example.com/success"),
		CancelURL:                stripe.String("https://example.com/cancel"),
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"order_id": orderId,
				"user_id":  claims.Subject,
				"products": string(jsonOutput),
			},
		},
	}
	sessionStripe, err := session.New(params)
	if err != nil {
		slog.Error("error creating Stripe checkout session", slog.String(logkey.TraceID, traceId), slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Stripe checkout session"})
		return
	}

	// Create the order
	err = h.o.CreateOrder(c.Request.Context(), orderId, claims.Subject, detailedItems, sessionStripe.AmountTotal)
	if err != nil {
		slog.Error("error creating order", slog.String(logkey.TraceID, traceId), slog.String("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Respond with Stripe session URL
	c.JSON(http.StatusOK, gin.H{"checkout_session_url": sessionStripe.URL})
}

func (h *Handler) fetchUserStripeCustomerID(ctx context.Context, authHeader, traceId string) (*UserServiceResponse, error) {
	address, port, err := consul.GetServiceAddress(h.client, "users")
	if err != nil {
		return nil, fmt.Errorf("service unavailable: %w", err)
	}

	httpQuery := fmt.Sprintf("http://%s:%d/users/stripe", address, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching user service: %w", err)
	}
	defer resp.Body.Close()

	var userServiceResponse UserServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&userServiceResponse); err != nil {
		return nil, fmt.Errorf("error decoding user service response: %w", err)
	}

	return &userServiceResponse, nil
}

func (h *Handler) fetchCartItems(ctx context.Context, authHeader, traceId string) ([]CartItem, error) {
	address, port, err := consul.GetServiceAddress(h.client, "cart")
	if err != nil {
		return nil, fmt.Errorf("cart service unavailable: %w", err)
	}

	cartURL := fmt.Sprintf("http://%s:%d/cart/items", address, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cartURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching cart service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching cart items: %s", resp.Status)
	}

	var cartResponse CartResponse
	if err := json.NewDecoder(resp.Body).Decode(&cartResponse); err != nil {
		return nil, fmt.Errorf("error decoding cart response: %w", err)
	}

	return cartResponse.Items, nil
}

// Fetch product details for cart items
func (h *Handler) fetchProductDetails(ctx context.Context, items []CartItem, authHeader, traceId string) ([]orders.DetailedCartItem, error) {
	address, port, err := consul.GetServiceAddress(h.client, "products")
	if err != nil {
		return nil, fmt.Errorf("product service unavailable: %w", err)
	}

	var detailedItems []orders.DetailedCartItem
	for _, item := range items {
		productURL := fmt.Sprintf("http://%s:%d/products/stock/%s", address, port, item.ProductID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, productURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Authorization", authHeader)

		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error fetching Product service: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error fetching product details for product %s: %s", item.ProductID, resp.Status)
		}

		var productResponse ProductServiceResponse
		if err := json.NewDecoder(resp.Body).Decode(&productResponse); err != nil {
			return nil, fmt.Errorf("error decoding product response: %w", err)
		}

		detailedItems = append(detailedItems, orders.DetailedCartItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     productResponse.Price,
			Stock:     productResponse.Stock,
			PriceID:   productResponse.PriceID,
		})
	}

	return detailedItems, nil
}
