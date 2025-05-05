package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"inventory-service/internal/service"
	"shared/pkg/event"
	"shared/pkg/nats"
)

type OrderConsumer struct {
	natsClient     *nats.Client
	productService service.ProductService
}

func NewOrderConsumer(natsClient *nats.Client, productService service.ProductService) *OrderConsumer {
	return &OrderConsumer{
		natsClient:     natsClient,
		productService: productService,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) error {
	log.Printf("Starting to consume messages from %s", event.OrderCreatedSubject)

	return c.natsClient.Subscribe(event.OrderCreatedSubject, func(data []byte) {
		startTime := time.Now()
		log.Printf("[CONSUMER] Received order created event at %s", startTime.Format(time.RFC3339))

		var orderEvent event.OrderCreatedEvent
		if err := json.Unmarshal(data, &orderEvent); err != nil {
			log.Printf("Error unmarshaling order event: %v", err)
			return
		}

		log.Printf("[CONSUMER] Processing order %s for user %s", orderEvent.OrderID, orderEvent.UserID)

		// Update inventory for each item
		for _, item := range orderEvent.Items {
			log.Printf("[CONSUMER] Reducing stock for product %s by %d units", item.ProductID, item.Quantity)

			// Get current product
			product, err := c.productService.GetProductByID(ctx, item.ProductID)
			if err != nil {
				log.Printf("Error getting product %s: %v", item.ProductID, err)
				continue
			}

			// Update stock
			product.Stock -= int(item.Quantity)
			if product.Stock < 0 {
				log.Printf("Warning: Product %s will have negative stock (%d)", item.ProductID, product.Stock)
			}

			if err := c.productService.UpdateProduct(ctx, product); err != nil {
				log.Printf("Error updating stock for product %s: %v", item.ProductID, err)
				continue
			}

			log.Printf("[CONSUMER] Successfully updated stock for product %s. New stock: %d", item.ProductID, product.Stock)
		}

		endTime := time.Now()
		duration := endTime.Sub(startTime)
		log.Printf("[CONSUMER] Finished processing order %s at %s (duration: %s)",
			orderEvent.OrderID, endTime.Format(time.RFC3339), duration)
	})
}
