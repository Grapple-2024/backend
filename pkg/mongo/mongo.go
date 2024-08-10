package mongo

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"gopkg.in/mgo.v2/bson"
)

type Client struct {
	*mongo.Client
}

func New(ctx context.Context, endpoint string) (*Client, error) {
	c, err := mongo.Connect(ctx, options.Client().ApplyURI(endpoint))
	if err != nil {
		return nil, err
	}

	if err := c.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &Client{c}, err
}

func Insert(ctx context.Context, collection *mongo.Collection, payload, result any) error {
	r, err := collection.InsertOne(ctx, payload, nil)
	if err != nil {
		return err
	}

	// Find the newly created document by ID, store the result in the result variable
	insertedID, ok := r.InsertedID.(primitive.ObjectID)
	if !ok {
		return fmt.Errorf("failed to convert id to primitive ObjectID: %s", r.InsertedID)
	}

	if err := FindByID(ctx, collection, insertedID.Hex(), result); err != nil {
		return err
	}

	return nil
}

func Find(ctx context.Context, collection *mongo.Collection, filter bson.M, result any) error {
	if err := collection.FindOne(ctx, filter).Decode(result); err != nil {
		return err
	}
	return nil
}

func FindByID(ctx context.Context, collection *mongo.Collection, id string, result any) error {
	// Convert the id to a primitive.ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Find the document by ID
	filter := bson.M{"_id": objID}
	if err := collection.FindOne(ctx, filter).Decode(result); err != nil {
		return err
	}

	return nil
}

func Update(ctx context.Context, c *mongo.Collection, update bson.M, filter bson.M, result any, opts *options.UpdateOptions) error {
	// update the record in mongo
	_, err := c.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	log.Debug().Msgf("Finding record with filter: %v", filter)
	if err := Find(ctx, c, filter, &result); err != nil {
		return err
	}
	log.Debug().Msgf("result : %+v", result)

	return nil
}

func UpdateByID(ctx context.Context, c *mongo.Collection, id string, payload any, result any, opts *options.UpdateOptions) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	log.Info().Msgf("Updating object ID: %v", objID)

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": payload}

	// update the record in mongo
	_, err = c.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	if err := FindByID(ctx, c, id, result); err != nil {
		return err
	}

	return nil
}

func Delete(ctx context.Context, collection *mongo.Collection, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": objID}
	opts := options.Delete().SetHint(bson.M{"_id": 1}) // use _id index to find object

	result, err := collection.DeleteOne(context.TODO(), filter, opts)
	if err != nil {
		return fmt.Errorf("could not delete record with id %q: %w", id, err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("could not delete any objects with id %s", id)
	}

	return nil
}

func Paginate(ctx context.Context, c *mongo.Collection, filter bson.M, page int, pageSize int, result any) error {
	// Calculate the number of documents to skip
	skip := (page - 1) * pageSize

	// Query options
	findOptions := options.Find()
	findOptions.SetSkip(int64(skip))
	findOptions.SetLimit(int64(pageSize))

	// Execute the query
	cursor, err := c.Find(ctx, filter, findOptions)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	// Collect and return results
	if err = cursor.All(ctx, result); err != nil {
		return err
	}

	return nil
}
