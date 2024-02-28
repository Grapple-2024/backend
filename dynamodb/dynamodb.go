package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/rs/zerolog/log"
)

type Client struct {
	*dynamodb.Client
	table string
	t     reflect.Type
}

func NewClient(endpoint, tableName string) (*Client, error) {
	log.Info().Msgf("Starting dynamo client with endpoint %s", endpoint)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint}, nil
			})),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: dynamodb.NewFromConfig(cfg),
		table:  tableName,
	}, nil
}

func (c *Client) GetByID(ctx context.Context, id string) (*dynamodb.QueryOutput, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(c.table),
		KeyConditions: map[string]types.Condition{
			"pk": {
				ComparisonOperator: types.ComparisonOperatorEq,
				AttributeValueList: []types.AttributeValue{
					key,
				},
			},
		},
	}
	result, err := c.Query(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	if result.Count == 0 {
		return nil, fmt.Errorf("no items found from query for ID: %v", id)
	}

	// err = attributevalue.UnmarshalListOfMaps(result.Items, data)
	// if err != nil {
	// 	return nil, err
	// }

	return result, nil
}

type Filter struct {
	FilterExpression          *string           `json:"filter,omitempty"`
	ExpressionAttributeNames  map[string]string `json:"expression_attribute_names,omitempty"`
	ExpressionAttributeValues map[string]any    `json:"expression_attribute_values"`
}

func (c *Client) Get(ctx context.Context, limit int32, exclusiveStartKey, indexName *string, filter *Filter) (*dynamodb.ScanOutput, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(c.table),
		Limit:     &limit,
	}

	if indexName != nil {
		input.IndexName = indexName
	}

	if filter != nil {
		avs, err := attributevalue.MarshalMap(filter.ExpressionAttributeValues)
		if err != nil {
			return nil, fmt.Errorf("error marshalling map to AVS %+v: %w", filter, err)
		}

		if filter.FilterExpression != nil {
			input.FilterExpression = filter.FilterExpression
		}
		if filter.ExpressionAttributeNames != nil {
			input.ExpressionAttributeNames = filter.ExpressionAttributeNames
		}
		if filter.ExpressionAttributeValues != nil {
			input.ExpressionAttributeValues = avs
		}
	}

	log.Info().Any("expressionAttributeValues", input.ExpressionAttributeValues).
		Any("expressionAttributeNames", input.ExpressionAttributeNames).
		Any("filter", input.FilterExpression).
		Any("table", *input.TableName).
		Msg("Scanning table")

	result, err := c.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error scanning table with input %v: %w", input, err)
	}

	return result, nil
}

func (c *Client) Insert(ctx context.Context, data any) (*dynamodb.PutItemOutput, error) {
	item, err := attributevalue.MarshalMap(data)
	if err != nil {
		return nil, err
	}
	log.Info().Any("item", item).Msgf("PUT::Table: %v", c.table)

	input := &dynamodb.PutItemInput{
		TableName: aws.String(c.table),
		Item:      item,
	}

	return c.PutItem(ctx, input)
}

func (c *Client) Update(ctx context.Context, key map[string]types.AttributeValue, expr *expression.Expression) (*dynamodb.UpdateItemOutput, error) {
	input := &dynamodb.UpdateItemInput{
		Key:                       key,
		TableName:                 aws.String(c.table),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              types.ReturnValue(*aws.String("ALL_NEW")),
	}

	res, err := c.UpdateItem(ctx, input)
	if err != nil {
		var smErr *smithy.OperationError
		if errors.As(err, &smErr) {
			var condCheckFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condCheckFailed) {
				return nil, err
			}
		}
		return nil, err
	}
	if res.Attributes == nil {
		return nil, fmt.Errorf("updateitem attributes are empty")
	}

	return res, nil
}

func (c *Client) Delete(ctx context.Context, key map[string]types.AttributeValue) (*dynamodb.DeleteItemOutput, error) {
	input := &dynamodb.DeleteItemInput{
		TableName:    aws.String(c.table),
		Key:          key,
		ReturnValues: types.ReturnValue(*aws.String("ALL_OLD")),
	}

	return c.DeleteItem(ctx, input)
}

func (c *Client) CreateTable(ctx context.Context, tableName *string, keySchema []types.KeySchemaElement, attributes []types.AttributeDefinition, gsis []types.GlobalSecondaryIndex) (*types.TableDescription, error) {
	var tableDesc *types.TableDescription
	table, err := c.Client.CreateTable(ctx, &dynamodb.CreateTableInput{
		GlobalSecondaryIndexes: gsis,
		AttributeDefinitions:   attributes,
		KeySchema:              keySchema,
		TableName:              tableName,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
	})
	if err != nil {
		log.Info().Err(err).Msgf("Couldn't create table %v", c.table)
	} else {
		log.Info().Msgf("Table created: %v", table.TableDescription)

	}

	return tableDesc, err
}
