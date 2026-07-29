package mongostore

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/n1ckerr0r/shortener/internal/link"
)

type Store struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func Open(ctx context.Context, uri, database, collection string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}

	return &Store{
		client:     client,
		collection: client.Database(database).Collection(collection),
	}, nil
}

func (s *Store) SaveClick(ctx context.Context, event link.ClickEvent) error {
	document := bson.M{
		"code":         event.Code,
		"original_url": event.OriginalURL,
		"clicked_at":   event.ClickedAt,
		"remote_addr":  event.RemoteAddr,
		"user_agent":   event.UserAgent,
		"stored_at":    time.Now().UTC(),
	}

	_, err := s.collection.InsertOne(ctx, document)
	return err
}

func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
