package kafka

import "time"

//<microservice>.<event-type>.<version> // topic naming

const TopicOrderPaid = `order-service.order-paid`
const ConsumerGroup = `product-service`

type OrderPaidEvent struct {
	OrderId   string    `json:"order_id"` // UUID
	ProductId string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"` // Timestamp of creation
}
