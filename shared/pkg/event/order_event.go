package event

import "time"

const (
	OrderCreatedSubject = "order.created"
)

type OrderCreatedEvent struct {
	OrderID   string      `json:"order_id"`
	UserID    string      `json:"user_id"`
	Status    string      `json:"status"`
	Total     float64     `json:"total"`
	Items     []OrderItem `json:"items"`
	CreatedAt time.Time   `json:"created_at"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	Quantity  int32   `json:"quantity"`
}
