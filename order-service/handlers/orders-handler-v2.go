package handlers

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"order-service/internal/auth"
	"order-service/pkg/ctxmanage"
	"order-service/pkg/logkey"
	"os"
	"time"

	pb "order-service/gen/proto"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

func (h *Handler) CheckoutV2(c *gin.Context) {
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
	cartItems, err := fetchCartItems(h.protoClient, claims.Subject)
	if err != nil {
		slog.Error("failed to fetch cart items", slog.String(logkey.TraceID, traceId))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart items"})
		return
	}

	//Fetch product product details

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

func fetchCartItems(client pb.CartItemServiceClient, userID string) ([]CartItem, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()

	req := &pb.GetCartDetailsRequest{UserId: userID}

	resp, err := client.GetCartDetails(ctx, req)

	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println(resp)

	items, err := ConvertResponseToItems(resp)
	if err != nil {
		return nil, err
	}

	return items, nil

}

func ConvertResponseToItems(resp *pb.GetCartDetailsResponse) ([]CartItem, error) {
	var items []CartItem
	for _, cartItem := range resp.CartItems {
		items = append(items, CartItem{
			ProductID: cartItem.ProductID,
			Quantity:  cartItem.Quantity,
		})
	}
	return items, nil
}
