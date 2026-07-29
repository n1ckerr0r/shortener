package kafkaevents

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/n1ckerr0r/shortener/internal/link"
)

const defaultClickTopic = "link.clicked"

type Publisher struct {
	writer *kafka.Writer
	events chan link.ClickEvent
	done   chan struct{}
	wg     sync.WaitGroup
	logger *slog.Logger
}

type PublisherOptions struct {
	Brokers []string
	Topic   string
	Workers int
	Buffer  int
	Logger  *slog.Logger
}

func NewPublisher(opts PublisherOptions) *Publisher {
	if opts.Topic == "" {
		opts.Topic = defaultClickTopic
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	if opts.Buffer <= 0 {
		opts.Buffer = 1024
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	publisher := &Publisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(opts.Brokers...),
			Topic:        opts.Topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireOne,
		},
		events: make(chan link.ClickEvent, opts.Buffer),
		done:   make(chan struct{}),
		logger: opts.Logger,
	}

	for i := 0; i < opts.Workers; i++ {
		publisher.wg.Add(1)
		go publisher.worker()
	}

	return publisher
}

func (p *Publisher) PublishClick(ctx context.Context, event link.ClickEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return errors.New("kafka publisher is closed")
	case p.events <- event:
		return nil
	default:
		return errors.New("kafka publisher buffer is full")
	}
}

func (p *Publisher) Close() error {
	close(p.done)
	p.wg.Wait()
	return p.writer.Close()
}

func (p *Publisher) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.done:
			return
		case event := <-p.events:
			p.write(event)
		}
	}
}

func (p *Publisher) write(event link.ClickEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("marshal_click_event_failed", slog.Any("error", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.Code),
		Value: payload,
		Time:  event.ClickedAt,
	}); err != nil {
		p.logger.Error("publish_click_event_failed", slog.String("code", event.Code), slog.Any("error", err))
	}
}
