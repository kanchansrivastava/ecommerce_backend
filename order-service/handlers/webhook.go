package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/smtp"
	"order-service/internal/orders"
	"order-service/internal/stores/kafka"
	"order-service/pkg/logkey"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
)

func (h *Handler) Webhook(c *gin.Context) {
	traceId := uuid.NewString()
	const MaxBodyBytes = int64(65536)

	// Limit the request body size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxBodyBytes)

	var event stripe.Event
	err := c.ShouldBindJSON(&event)
	if err != nil {
		slog.Error("Failed to bind JSON", slog.Any("error", err.Error()))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			slog.Error("Failed to unmarshal JSON", slog.Any("error", err.Error()))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		slog.Info("Payment Intent Succeeded", slog.Any("paymentIntent ID", paymentIntent.ID))
		orderId := paymentIntent.Metadata["order_id"]
		productsJson := paymentIntent.Metadata["products"] // JSON string
		userID := paymentIntent.Metadata["user_id"]
		slog.Info("Metadata received", slog.String(logkey.TraceID, traceId), slog.String("OrderID", orderId), slog.String("UserID", userID))

		// Unmarshal products into a slice of map[string]interface{} (assuming the products are an array of items)
		var products []map[string]interface{}
		err = json.Unmarshal([]byte(productsJson), &products)
		if err != nil {
			slog.Error("Failed to unmarshal products", slog.Any("error", err.Error()))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Goroutine to handle each product
		go func() {
			for _, item := range products {
				productID := item["product_id"].(string)    // Assuming the product ID is stored as "price_id" in the JSON
				quantity := int(item["quantity"].(float64)) // Assuming quantity is stored as a number
				// Create OrderPaidEvent for each product
				jsonData, err := json.Marshal(kafka.OrderPaidEvent{
					OrderId:   orderId,
					ProductId: productID,
					Quantity:  quantity,
					CreatedAt: time.Now().UTC(),
				})
				if err != nil {
					slog.Error("Failed to marshal OrderPaidEvent", slog.Any("error", err.Error()))
					return
				}

				// Produce the message to Kafka
				key := []byte(orderId)
				err = h.k.ProduceMessage(kafka.TopicOrderPaid, key, jsonData)
				if err != nil {
					slog.Error("Failed to produce message", slog.Any("error", err.Error()))
					return
				}
				slog.Info("Message produced", slog.Any("data", string(jsonData)))
			}
		}()

		// Update order status to 'paid' in the database
		ctx := c.Request.Context()
		err = h.o.UpdateOrder(ctx, orderId, orders.StatusPaid, paymentIntent.ID)
		if err != nil {
			slog.Error("Failed to update order", slog.Any("error", err.Error()))
			return
		}
		sendOrderConfirmationEmail("kanchansrivastava1991@gmail.com", orderId)
		// Respond with HTTP status OK
		c.Status(http.StatusOK)

	default:
		slog.Info("Unhandled event type", slog.String("event_type", string(event.Type)))
		c.JSON(http.StatusOK, gin.H{
			"message": "Event type not handled",
			"event":   event.Type,
		})
	}
}

func sendOrderConfirmationEmail(to string, orderId string) error {
	smtpHost := "smtp.mailtrap.io" // Mailtrap SMTP server
	smtpPort := "587"              // Mailtrap SMTP port
	username := "f2763311c38489"   // Your Mailtrap username
	password := "286a588d2e4f82"   // Your Mailtrap password

	subject := "Order Confirmation"
	body := fmt.Sprintf("Thank you for your order! Your order ID is %s. We are processing it now.", orderId)

	from := "no-reply@example.com"
	message := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", username, password, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, message)
	if err != nil {
		log.Printf("Failed to send email: %v", err)
		return err
	}

	log.Printf("Email sent successfully to %s", to)
	return nil
}
