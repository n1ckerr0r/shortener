package kafkaevents

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"

	"github.com/n1ckerr0r/shortener/internal/link"
)

type Consumer struct {
	reader *kafka.Reader
}

type ConsumerOptions struct {
	Brokers []string
	Topic   string
	GroupID string
}

func NewConsumer(opts ConsumerOptions) *Consumer {
	if opts.Topic == "" {
		opts.Topic = defaultClickTopic
	}
	if opts.GroupID == "" {
		opts.GroupID = "shortener-analytics"
	}

	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: opts.Brokers,
			Topic:   opts.Topic,
			GroupID: opts.GroupID,
		}),
	}
}

func (c *Consumer) ReadClick(ctx context.Context) (link.ClickEvent, error) {
	message, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return link.ClickEvent{}, err
	}

	var event link.ClickEvent
	if err = json.Unmarshal(message.Value, &event); err != nil {
		return link.ClickEvent{}, err
	}

	return event, nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
