package mongo

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database = mongo.Database

var ErrNoDocuments = mongo.ErrNoDocuments

type Client struct {
	*mongo.Client
	*mongo.Database
}

// NewClient creates a new mongo client and pings it to make sure the connection was successful.
// Params:
//   - ctx: the context to use when connecting and pinging the database
//   - uri: the mongo server connection string
//
// Returns:
//   - *Client: the mongo client
//   - error: an error, if any
//
// Usage:
//
//	NewClient(context.TODO(), "mongodb:/username:password@mymongoserver.com:6379/")
func NewClient(ctx context.Context, uri, dbName string) (*Client, error) {
	// uri := fmt.Sprintf("mongodb://%s:%s@%s:%s/", *mongoUser, *mongoPass, *mongoHost, *mongoPort)
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetRegistry(bson.NewRegistry()))
	if err != nil {
		return nil, err
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return &Client{
		Client:   mongoClient,
		Database: mongoClient.Database(dbName),
	}, nil
}

func (c *Client) Find(ctx context.Context, collection string) ([]map[string]any, error) {
	q := bson.M{}

	cursor, err := c.Collection(collection).Find(ctx, q)
	if err != nil {
		return nil, err
	}

	// defer closing the cursor
	defer cursor.Close(ctx)

	result := []map[string]any{}

	// cursor.Next() returns a boolean, if false there are no more items and loop will break
	for cursor.Next(ctx) {
		// Initiate a Recipe type to write decoded data to
		data := map[string]any{}

		// Decode the data at the current pointer and write it to data
		if err := cursor.Decode(data); err != nil {
			return nil, err
		}
		result = append(result, data)
	}
	log.Info().Msgf("result: %v", result)

	// Check if the cursor has any errors
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) FindOne(ctx context.Context, collection string, id string, obj any) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	q := bson.M{"_id": objectID}

	return c.Collection(collection).FindOne(ctx, q).Decode(&obj)
}
