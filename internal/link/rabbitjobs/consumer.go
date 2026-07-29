package rabbitjobs

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/n1ckerr0r/shortener/internal/link"
)

type Consumer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	queue      string
	deliveries <-chan amqp.Delivery
}

type URLCheckDelivery struct {
	Job  link.URLCheckJob
	Ack  func() error
	Nack func(requeue bool) error
}

func NewConsumer(url, queue string) (*Consumer, error) {
	if queue == "" {
		queue = defaultURLCheckQueue
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err = channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}
	if err = channel.Qos(8, 0, false); err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	deliveries, err := channel.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		_ = channel.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Consumer{
		conn:       conn,
		channel:    channel,
		queue:      queue,
		deliveries: deliveries,
	}, nil
}

func (c *Consumer) Jobs(ctx context.Context) <-chan URLCheckDelivery {
	jobs := make(chan URLCheckDelivery)

	go func() {
		defer close(jobs)
		for {
			select {
			case <-ctx.Done():
				return
			case delivery, ok := <-c.deliveries:
				if !ok {
					return
				}

				var job link.URLCheckJob
				if err := json.Unmarshal(delivery.Body, &job); err != nil {
					_ = delivery.Nack(false, false)
					continue
				}
				deliveryCopy := delivery

				urlCheckDelivery := URLCheckDelivery{
					Job: job,
					Ack: func() error {
						return deliveryCopy.Ack(false)
					},
					Nack: func(requeue bool) error {
						return deliveryCopy.Nack(false, requeue)
					},
				}

				select {
				case <-ctx.Done():
					return
				case jobs <- urlCheckDelivery:
				}
			}
		}
	}()

	return jobs
}

func (c *Consumer) Close() error {
	if err := c.channel.Close(); err != nil {
		_ = c.conn.Close()
		return err
	}

	return c.conn.Close()
}
