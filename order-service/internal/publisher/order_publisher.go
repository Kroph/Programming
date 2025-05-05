package publisher

import (
	"log"
	"time"

	"order-service/internal/domain"

	"github.com/Kroph/Programming/shared/pkg/event"
	"github.com/Kroph/Programming/shared/pkg/nats"
)

type OrderPublisher interface {
	PublishOrderCreated(order domain.Order) error
}

type orderPublisher struct {
	natsClient *nats.Client
}

func NewOrderPublisher(natsClient *nats.Client) OrderPublisher {
	return &orderPublisher{
		natsClient: natsClient,
	}
}

func (p *orderPublisher) PublishOrderCreated(order domain.Order) error {
	items := make([]event.OrderItem, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, event.OrderItem{
			ProductID: item.ProductID,
			Name:      item.Name,
			Price:     item.Price,
			Quantity:  int32(item.Quantity),
		})
	}

	orderEvent := event.OrderCreatedEvent{
		OrderID:   order.ID,
		UserID:    order.UserID,
		Status:    string(order.Status),
		Total:     order.Total,
		Items:     items,
		CreatedAt: time.Now(),
	}

	log.Printf("Publishing order created event for order %s", order.ID)
	return p.natsClient.Publish(event.OrderCreatedSubject, orderEvent)
}
