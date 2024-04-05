package dynamodb

import (
	"context"
	"encoding/json"
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
}

type Key struct {
	PK string `dynamodbav:"pk,omitempty" json:"pk,omitempty"`
	SK string `dynamodbav:"sk,omitempty" json:"sk,omitempty"`
}

type GetResponse struct {
	Data          any    `json:"data"`
	Count         int32  `json:"count"`
	ScannedCount  int32  `json:"scanned_count"`
	LastEvaluated string `json:"lastEvaluatedKey"`
}

func MarshalResponse(sortKey *string, limit, count, scannedCount int32, lastEvaluatedKey map[string]types.AttributeValue, items []map[string]types.AttributeValue, data any) (*GetResponse, error) {
	resp := &GetResponse{
		Data:         data,
		Count:        count,
		ScannedCount: scannedCount,
	}
	if len(items) <= int(limit) {
		if err := attributevalue.UnmarshalListOfMaps(items, data); err != nil {
			return nil, err
		}
		lastEvaledMap := map[string]any{}
		if err := attributevalue.UnmarshalMap(lastEvaluatedKey, &lastEvaledMap); err != nil {
			return nil, err
		}

		// marshal map[string]types.AttributeValue -> json string
		bytes, err := json.Marshal(lastEvaledMap)
		if err != nil {
			return nil, err
		}
		resp.LastEvaluated = string(bytes)
		return resp, nil
	}

	// filter results
	// log.Info().Msgf("%d > %d is true", len(items), int(limit))
	items = items[:limit]
	if err := attributevalue.UnmarshalListOfMaps(items, data); err != nil {
		return nil, err
	}

	resp.Count = limit
	lastEvaled := map[string]any{
		"pk":     items[limit-1]["pk"].(*types.AttributeValueMemberS).Value,
		"dummy":  "dumb",
		*sortKey: items[limit-1][*sortKey].(*types.AttributeValueMemberS).Value,
	}

	jsonBytes, err := json.Marshal(lastEvaled)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("Last Evaluated: %s", string(jsonBytes))
	resp.LastEvaluated = string(jsonBytes)

	return resp, nil
}

func NewClient(endpoint string) (*Client, error) {
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
	}, nil
}

func (c *Client) GetByID(ctx context.Context, table, id string) (*dynamodb.QueryOutput, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(table),
		KeyConditions: map[string]types.Condition{
			"pk": {
				ComparisonOperator: types.ComparisonOperatorEq,
				AttributeValueList: []types.AttributeValue{
					key,
				},
			},
		},
	}

	log.Info().Msgf("Query for %s: %+v", *queryInput.TableName, queryInput)
	result, err := c.Query(ctx, queryInput)
	if err != nil {
		return nil, err
	}

	if result.Count == 0 {
		log.Warn().Msgf("result count is 0 for query for %v by pk=%v", *queryInput.TableName, key)
		return nil, fmt.Errorf("no items found from query for ID: %v", id)
	}

	// err = attributevalue.UnmarshalListOfMaps(result.Items, data)
	// if err != nil {
	// 	return nil, err
	// }

	return result, nil
}

type Filter struct {
	KeyConditionExpression    *string           `json:"key_condition_expression,omitempty"`
	FilterExpression          *string           `json:"filter,omitempty"`
	ExpressionAttributeNames  map[string]string `json:"expression_attribute_names,omitempty"`
	ExpressionAttributeValues map[string]any    `json:"expression_attribute_values"`
}

func (c *Client) QueryPage(ctx context.Context, table string, limit int32, startKey map[string]types.AttributeValue, indexName *string, filter *Filter, ascending bool) (*dynamodb.QueryOutput, error) {
	input := &dynamodb.QueryInput{
		TableName:        aws.String(table),
		Limit:            &limit,
		ScanIndexForward: aws.Bool(ascending),
	}

	if len(startKey) != 0 {
		input.ExclusiveStartKey = startKey
	}

	if indexName != nil {
		input.IndexName = indexName
	}

	if filter != nil {
		avs, err := attributevalue.MarshalMap(filter.ExpressionAttributeValues)
		if err != nil {
			return nil, fmt.Errorf("error marshalling map to AVS %+v: %w", filter, err)
		}

		if filter.KeyConditionExpression != nil {
			input.KeyConditionExpression = filter.KeyConditionExpression
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
		Any("keyConditionExpression", input.KeyConditionExpression).
		Any("expressionAttributeNames", input.ExpressionAttributeNames).
		Any("filter", input.FilterExpression).
		Any("table", *input.TableName).
		Any("ascending", *input.ScanIndexForward).
		Msg("Scanning table")

	result, err := c.Query(ctx, input)
	if temp := new(types.ResourceNotFoundException); errors.As(err, &temp) {
		// ignore 404 not found error and just return empty slice
		return &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{},
		}, nil

	} else if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) Get(ctx context.Context, table string, limit int32, exclusiveStartKey, indexName *string, filter *Filter) (*dynamodb.ScanOutput, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String(table),
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

func (c *Client) Insert(ctx context.Context, table string, data any) (*dynamodb.PutItemOutput, error) {
	item, err := attributevalue.MarshalMap(data)
	if err != nil {
		return nil, err
	}
	log.Info().Any("item", item).Msgf("PUT::Table: %v", table)

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(pk)"),
	}
	o, err := c.PutItem(ctx, input)
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return nil, fmt.Errorf("object already exists in table: %v", err)
		}
		return nil, err
	}

	return o, nil
}

func (c *Client) Update(ctx context.Context, table string, key map[string]types.AttributeValue, expr *expression.Expression) (*dynamodb.UpdateItemOutput, error) {
	input := &dynamodb.UpdateItemInput{
		Key:                       key,
		TableName:                 aws.String(table),
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

func (c *Client) Delete(ctx context.Context, table string, key map[string]types.AttributeValue) (*dynamodb.DeleteItemOutput, error) {
	input := &dynamodb.DeleteItemInput{
		TableName:    aws.String(table),
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
		log.Info().Err(err).Msgf("Couldn't create table %v", err)
	} else {
		log.Info().Msgf("Table created: %v", table.TableDescription)
	}

	return tableDesc, err
}

type Condition struct {
	Operator string
	Value    any
}

func BuildExpression(conditions map[string]Condition) *expression.ConditionBuilder {
	var builder *expression.ConditionBuilder
	for field, condition := range conditions {
		if reflect.TypeOf(condition.Value).String() == "[]string" && len(condition.Value.([]string)) == 0 {
			log.Info().Msgf("skipping empty []string condition")

			continue
		} else if reflect.TypeOf(condition.Value).String() == "string" && len(condition.Value.(string)) == 0 {
			log.Info().Msgf("skipping empty string condition")
			continue
		}

		var cond expression.ConditionBuilder
		switch condition.Operator {
		case "Equal":
			cond = expression.Name(field).Equal(expression.Value(condition.Value))
		case "Contains":
			cond = expression.Contains(expression.Name(field), condition.Value)
		case "ContainsOr":
			vals := condition.Value.([]string)
			for i, v := range vals {
				if i != 0 {
					cond = cond.Or(expression.Contains(expression.Name(field), v))
					continue
				}
				cond = expression.Contains(expression.Name(field), v)
			}

			log.Info().Msgf("Built containsOr condition: %+v", cond)

		case "StringIn":
			values := []expression.OperandBuilder{}
			vals := condition.Value.([]string)
			for _, v := range vals {
				values = append(values, expression.Value(v))
			}
			log.Info().Msgf("Filtering objects where string field %q is equal to one of the values in the set %v", field, values)
			cond = expression.In(expression.Name(field), values[0], values[1:]...)
		default:
			cond = expression.Name(field).Equal(expression.Value(condition.Value))
		}

		if builder == nil {
			builder = &cond
			continue
		}

		*builder = builder.And(cond)
	}

	return builder
}
