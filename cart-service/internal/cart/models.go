package cart

type Cart struct {
}

type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

type CartResponse struct {
	Items []CartItem `json:"items"`
}
