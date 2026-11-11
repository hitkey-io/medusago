package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// oneDayInSeconds mirrors connect-dynamodb's default session lifetime.
const oneDayInSeconds = 86400

// DynamoDBAPI is the part of the DynamoDB client the store uses.
type DynamoDBAPI interface {
	DescribeTable(ctx context.Context, in *dynamodb.DescribeTableInput, opts ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	CreateTable(ctx context.Context, in *dynamodb.CreateTableInput, opts ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
	GetItem(ctx context.Context, in *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, in *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	UpdateItem(ctx context.Context, in *dynamodb.UpdateItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, in *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Scan(ctx context.Context, in *dynamodb.ScanInput, opts ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// DynamoDBSpecialKey mirrors an entry of the specialKeys option: a top level
// session property also stored as its own item attribute.
type DynamoDBSpecialKey struct {
	Name string
	Type string
}

// DynamoDBStore mirrors connect-dynamodb 3 (DynamoDBStore): one item per
// session in the table (default "sessions"), keyed by hashKey (default
// "id") = prefix (default "sess:") + sid, with the attributes expires (N,
// seconds), type "connect-session" and sess (the JSON session).
type DynamoDBStore struct {
	client                      DynamoDBAPI
	table, prefix, hashKey      string
	readCapacity, writeCapacity int64
	specialKeys                 []DynamoDBSpecialKey
	skipThrowMissingSpecialKeys bool

	mu          sync.Mutex
	initialized bool
	stopReap    chan struct{}
}

// NewDynamoDBStore mirrors `new DynamoDBStore({ ...dynamodbOptions, client:
// new DynamoDBClient(dynamodbOptions.clientOptions) })` of the express
// loader. A non-nil client replaces the one built from clientOptions.
func NewDynamoDBStore(options map[string]any, client DynamoDBAPI) (*DynamoDBStore, error) {
	if options == nil {
		options = map[string]any{}
	}
	s := &DynamoDBStore{
		prefix:        "sess:",
		hashKey:       "id",
		readCapacity:  5,
		writeCapacity: 5,
		table:         "sessions",
	}
	s.initialized = truthy(options["initialized"])
	if v, ok := options["prefix"]; ok && v != nil {
		s.prefix = fmt.Sprint(v)
	}
	if v, ok := options["hashKey"]; ok && v != nil {
		s.hashKey = fmt.Sprint(v)
	}
	if v, ok := options["readCapacityUnits"]; ok && v != nil {
		s.readCapacity = parseIntJS(v)
	}
	if v, ok := options["writeCapacityUnits"]; ok && v != nil {
		s.writeCapacity = parseIntJS(v)
	}
	if keys, ok := options["specialKeys"].([]any); ok {
		for _, k := range keys {
			if m, ok := k.(map[string]any); ok {
				s.specialKeys = append(s.specialKeys, DynamoDBSpecialKey{Name: fmt.Sprint(m["name"]), Type: fmt.Sprint(m["type"])})
			}
		}
	}
	// upstream quirk: the flag is `!!options.specialKeys` when it is set
	if v, ok := options["skipThrowMissingSpecialKeys"]; ok && v != nil {
		s.skipThrowMissingSpecialKeys = options["specialKeys"] != nil
	}
	if t, ok := options["table"].(string); ok && t != "" {
		s.table = t
	}
	if client == nil {
		clientOptions, _ := options["clientOptions"].(map[string]any)
		c, err := newDynamoDBClient(clientOptions)
		if err != nil {
			return nil, err
		}
		client = c
	}
	s.client = client
	if interval := parseIntJS(options["reapInterval"]); interval > 0 {
		s.stopReap = make(chan struct{})
		go func() {
			ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					_ = s.Reap()
				case <-s.stopReap:
					return
				}
			}
		}()
	}
	return s, nil
}

// newDynamoDBClient mirrors new DynamoDBClient(clientOptions): the default
// AWS configuration (environment, shared files) with the region, endpoint
// and credentials of the options.
func newDynamoDBClient(o map[string]any) (*dynamodb.Client, error) {
	var loaders []func(*awsconfig.LoadOptions) error
	if region, ok := o["region"].(string); ok && region != "" {
		loaders = append(loaders, awsconfig.WithRegion(region))
	}
	if c, ok := o["credentials"].(map[string]any); ok {
		id, _ := c["accessKeyId"].(string)
		secret, _ := c["secretAccessKey"].(string)
		token, _ := c["sessionToken"].(string)
		loaders = append(loaders, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(id, secret, token)))
	}
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), loaders...)
	if err != nil {
		return nil, err
	}
	var opts []func(*dynamodb.Options)
	if endpoint, ok := o["endpoint"].(string); ok && endpoint != "" {
		opts = append(opts, func(do *dynamodb.Options) { do.BaseEndpoint = aws.String(endpoint) })
	}
	return dynamodb.NewFromConfig(cfg, opts...), nil
}

// parseIntJS mirrors parseInt(v, 10) (0 for NaN).
func parseIntJS(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case string:
		n := 0
		for n < len(t) && (t[n] >= '0' && t[n] <= '9' || n == 0 && (t[n] == '-' || t[n] == '+')) {
			n++
		}
		i, _ := strconv.ParseInt(t[:n], 10, 64)
		return i
	}
	return 0
}

// initialize mirrors DynamoDBStore.initialize: the table is described and
// created when missing.
func (s *DynamoDBStore) initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initialized {
		return nil
	}
	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(s.table)})
	var notFound *types.ResourceNotFoundException
	if errors.As(err, &notFound) {
		_, err = s.client.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName:            aws.String(s.table),
			AttributeDefinitions: []types.AttributeDefinition{{AttributeName: aws.String(s.hashKey), AttributeType: types.ScalarAttributeTypeS}},
			KeySchema:            []types.KeySchemaElement{{AttributeName: aws.String(s.hashKey), KeyType: types.KeyTypeHash}},
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(s.readCapacity),
				WriteCapacityUnits: aws.Int64(s.writeCapacity),
			},
		})
	}
	if err != nil {
		return err
	}
	s.initialized = true
	return nil
}

func (s *DynamoDBStore) key(sid string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{s.hashKey: &types.AttributeValueMemberS{Value: s.prefix + sid}}
}

// Get mirrors DynamoDBStore.get / getParsedSession: a consistent read; an
// item without sess or expired yields no session.
func (s *DynamoDBStore) Get(sid string) ([]byte, bool, error) {
	ctx := context.Background()
	if err := s.initialize(ctx); err != nil {
		return nil, false, err
	}
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(s.table), Key: s.key(sid), ConsistentRead: aws.Bool(true)})
	if err != nil {
		return nil, false, err
	}
	now := float64(time.Now().Unix())
	sess, ok := out.Item["sess"].(*types.AttributeValueMemberS)
	if !ok || sess.Value == "" {
		return nil, false, nil
	}
	if exp, ok := out.Item["expires"].(*types.AttributeValueMemberN); ok {
		if e, err := strconv.ParseFloat(exp.Value, 64); err == nil && now >= e {
			return nil, false, nil
		}
	}
	if !json.Valid([]byte(sess.Value)) {
		return nil, false, nil
	}
	return []byte(sess.Value), true, nil
}

// expiresValue mirrors getExpiresValue: now (seconds) + cookie.maxAge / 1000
// (the remaining milliseconds of the cookie), or one day without expiry.
func expiresValue(expires time.Time) string {
	nowMs := time.Now().UnixMilli()
	now := math.Floor(float64(nowMs) / 1000)
	if expires.IsZero() {
		return totals.JSNumberToString(now + oneDayInSeconds)
	}
	return totals.JSNumberToString(now + float64(expires.UnixMilli()-nowMs)/1000)
}

// Set mirrors DynamoDBStore.set: PutItem with the session JSON.
func (s *DynamoDBStore) Set(sid string, data []byte, expires time.Time) error {
	ctx := context.Background()
	if err := s.initialize(ctx); err != nil {
		return err
	}
	item := s.key(sid)
	item["expires"] = &types.AttributeValueMemberN{Value: expiresValue(expires)}
	item["type"] = &types.AttributeValueMemberS{Value: "connect-session"}
	item["sess"] = &types.AttributeValueMemberS{Value: string(data)}
	if len(s.specialKeys) > 0 {
		var sess map[string]any
		_ = json.Unmarshal(data, &sess)
		var missing []string
		for _, k := range s.specialKeys {
			v, ok := sess[k.Name]
			if !ok {
				missing = append(missing, k.Name)
				continue
			}
			var attr types.AttributeValue
			switch k.Type {
			case "N":
				b, _ := json.Marshal(v)
				attr = &types.AttributeValueMemberN{Value: string(b)}
			case "BOOL":
				b, _ := v.(bool)
				attr = &types.AttributeValueMemberBOOL{Value: b}
			default:
				attr = &types.AttributeValueMemberS{Value: fmt.Sprint(v)}
			}
			item[k.Name] = attr
		}
		if !s.skipThrowMissingSpecialKeys && len(missing) > 0 {
			b, _ := json.Marshal(missing)
			return errors.New("Session missing special keys" + string(b))
		}
	}
	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(s.table), Item: item})
	return err
}

// Touch mirrors DynamoDBStore.touch: the expires attribute is updated.
func (s *DynamoDBStore) Touch(sid string, _ []byte, expires time.Time) error {
	ctx := context.Background()
	if err := s.initialize(ctx); err != nil {
		return err
	}
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(s.table),
		Key:                       s.key(sid),
		UpdateExpression:          aws.String("set expires = :e"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":e": &types.AttributeValueMemberN{Value: expiresValue(expires)}},
		ReturnValues:              types.ReturnValueUpdatedNew,
	})
	return err
}

// Destroy mirrors DynamoDBStore.destroy.
func (s *DynamoDBStore) Destroy(sid string) error {
	ctx := context.Background()
	if err := s.initialize(ctx); err != nil {
		return err
	}
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(s.table), Key: s.key(sid)})
	return err
}

// Reap mirrors DynamoDBStore.reap: the expired sessions are deleted.
func (s *DynamoDBStore) Reap() error {
	ctx := context.Background()
	if err := s.initialize(ctx); err != nil {
		return err
	}
	in := &dynamodb.ScanInput{
		TableName: aws.String(s.table),
		ScanFilter: map[string]types.Condition{"expires": {
			AttributeValueList: []types.AttributeValue{&types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)}},
			ComparisonOperator: types.ComparisonOperatorLt,
		}},
		AttributesToGet: []string{s.hashKey},
	}
	for {
		out, err := s.client.Scan(ctx, in)
		if err != nil {
			return err
		}
		for _, item := range out.Items {
			if id, ok := item[s.hashKey].(*types.AttributeValueMemberS); ok && len(id.Value) >= len(s.prefix) {
				_ = s.Destroy(id.Value[len(s.prefix):])
			}
		}
		if out.LastEvaluatedKey == nil {
			return nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

// Close mirrors clearInterval (stops the reaping).
func (s *DynamoDBStore) Close() {
	if s.stopReap != nil {
		close(s.stopReap)
		s.stopReap = nil
	}
}
