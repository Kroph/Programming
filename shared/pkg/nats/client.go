package nats

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type Client struct {
	conn *nats.Conn
}

func NewClient(url string) (*Client, error) {
	conn, err := nats.Connect(
		url,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(10),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.DisconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS disconnected")
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
}

func (c *Client) Publish(subject string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.conn.Publish(subject, jsonData)
}

func (c *Client) Subscribe(subject string, handler func([]byte)) error {
	_, err := c.conn.Subscribe(subject, func(msg *nats.Msg) {
		handler(msg.Data)
	})
	return err
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
