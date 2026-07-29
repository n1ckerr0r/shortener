package rabbitjobs

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/n1ckerr0r/shortener/internal/link"
)

const defaultURLCheckQueue = "url.check"

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
}

func NewPublisher(url, queue string) (*Publisher, error) {
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

	return &Publisher{
		conn:    conn,
		channel: channel,
		queue:   queue,
	}, nil
}

func (p *Publisher) PublishURLCheck(ctx context.Context, job link.URLCheckJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(ctx, "", p.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         payload,
	})
}

func (p *Publisher) Close() error {
	if err := p.channel.Close(); err != nil {
		_ = p.conn.Close()
		return err
	}

	return p.conn.Close()
}
