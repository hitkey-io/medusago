package http

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// TestDynamoDBStore runs against a local DynamoDB
// (MEDUSA_TEST_DYNAMODB_ENDPOINT, e.g. docker amazon/dynamodb-local):
//
//	MEDUSA_TEST_DYNAMODB_ENDPOINT=http://localhost:18471 go test ./packages/core/framework/src/http/ -run DynamoDB
//
// MEDUSA_TEST_DYNAMODB_READ=<table>:<sid> also reads a session written by
// connect-dynamodb (interoperability check).
func TestDynamoDBStore(t *testing.T) {
	endpoint := os.Getenv("MEDUSA_TEST_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		t.Skip("MEDUSA_TEST_DYNAMODB_ENDPOINT not set")
	}
	clientOptions := map[string]any{
		"endpoint":    endpoint,
		"region":      "us-east-1",
		"credentials": map[string]any{"accessKeyId": "local", "secretAccessKey": "local"},
	}
	table := fmt.Sprintf("go_store_test_%d", time.Now().UnixNano())
	store, err := NewDynamoDBStore(map[string]any{"table": table, "prefix": "shop:", "clientOptions": clientOptions}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client := store.client.(*dynamodb.Client)

	// a missing session; the table is created on first use
	if _, ok, err := store.Get("missing"); err != nil || ok {
		t.Fatalf("missing: %v %v", ok, err)
	}
	desc, err := client.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{TableName: aws.String(table)})
	if err != nil || *desc.Table.KeySchema[0].AttributeName != "id" || *desc.Table.ProvisionedThroughput.ReadCapacityUnits != 5 {
		t.Fatalf("table: %v", err)
	}

	data := []byte(`{"cookie":{"originalMaxAge":36000000,"expires":"2030-01-01T00:00:00.000Z","httpOnly":true,"path":"/"},"auth_context":{"actor_id":"user_1"}}`)
	expires := time.Now().Add(10 * time.Hour)
	if err := store.Set("abc", data, expires); err != nil {
		t.Fatal(err)
	}
	item, err := client.GetItem(context.Background(), &dynamodb.GetItemInput{TableName: aws.String(table), Key: map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "shop:abc"}}})
	if err != nil {
		t.Fatal(err)
	}
	if v := item.Item["type"].(*types.AttributeValueMemberS).Value; v != "connect-session" {
		t.Fatalf("type %q", v)
	}
	if v := item.Item["sess"].(*types.AttributeValueMemberS).Value; v != string(data) {
		t.Fatalf("sess %q", v)
	}
	exp, _ := strconv.ParseFloat(item.Item["expires"].(*types.AttributeValueMemberN).Value, 64)
	if want := float64(expires.Unix()); exp < want-2 || exp > want+2 {
		t.Fatalf("expires %v, want about %v", exp, want)
	}
	got, ok, err := store.Get("abc")
	if err != nil || !ok || string(got) != string(data) {
		t.Fatalf("get: %s %v %v", got, ok, err)
	}

	// touch moves expires
	if err := store.Touch("abc", data, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	item, _ = client.GetItem(context.Background(), &dynamodb.GetItemInput{TableName: aws.String(table), Key: map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "shop:abc"}}})
	exp, _ = strconv.ParseFloat(item.Item["expires"].(*types.AttributeValueMemberN).Value, 64)
	if want := float64(time.Now().Add(time.Hour).Unix()); exp < want-2 || exp > want+2 {
		t.Fatalf("touched expires %v", exp)
	}

	// an expired item is no session; reap deletes it
	if err := store.Set("old", data, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.Get("old"); ok {
		t.Fatal("expired session returned")
	}
	if err := store.Reap(); err != nil {
		t.Fatal(err)
	}
	item, _ = client.GetItem(context.Background(), &dynamodb.GetItemInput{TableName: aws.String(table), Key: map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "shop:old"}}})
	if len(item.Item) != 0 {
		t.Fatal("expired session not reaped")
	}

	// a session cookie without expiry lives one day
	if err := store.Set("day", data, time.Time{}); err != nil {
		t.Fatal(err)
	}
	item, _ = client.GetItem(context.Background(), &dynamodb.GetItemInput{TableName: aws.String(table), Key: map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: "shop:day"}}})
	exp, _ = strconv.ParseFloat(item.Item["expires"].(*types.AttributeValueMemberN).Value, 64)
	if want := float64(time.Now().Unix() + oneDayInSeconds); exp < want-2 || exp > want+2 {
		t.Fatalf("one day expires %v", exp)
	}

	if err := store.Destroy("abc"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.Get("abc"); ok {
		t.Fatal("destroyed session returned")
	}

	// special keys must be present
	special, _ := NewDynamoDBStore(map[string]any{"table": table, "initialized": true, "specialKeys": []any{map[string]any{"name": "user_id", "type": "S"}}, "clientOptions": clientOptions}, nil)
	if err := special.Set("s1", data, expires); err == nil || err.Error() != `Session missing special keys["user_id"]` {
		t.Fatalf("special keys: %v", err)
	}

	if spec := os.Getenv("MEDUSA_TEST_DYNAMODB_READ"); spec != "" {
		var readTable, sid string
		for i := len(spec) - 1; i >= 0; i-- {
			if spec[i] == ':' {
				readTable, sid = spec[:i], spec[i+1:]
				break
			}
		}
		reader, _ := NewDynamoDBStore(map[string]any{"table": readTable, "clientOptions": clientOptions}, nil)
		got, ok, err := reader.Get(sid)
		if err != nil || !ok {
			t.Fatalf("connect-dynamodb session %s: %v %v", spec, ok, err)
		}
		t.Logf("read %s", got)
		if err := reader.Set(sid+"-go", got, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
}
