package dynamodb

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type mockDynamo struct {
	mu sync.Mutex

	getItemOut    *dynamodb.GetItemOutput
	getItemErr    error
	lastGetItemIn *dynamodb.GetItemInput

	putItemErr    error
	lastPutItemIn *dynamodb.PutItemInput

	deleteItemOut *dynamodb.DeleteItemOutput
	deleteItemErr error
	lastDeleteIn  *dynamodb.DeleteItemInput

	queryOut    *dynamodb.QueryOutput
	queryErr    error
	lastQueryIn *dynamodb.QueryInput
}

func (m *mockDynamo) GetItem(ctx context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastGetItemIn = params
	if m.getItemErr != nil {
		return nil, m.getItemErr
	}
	return m.getItemOut, nil
}

func (m *mockDynamo) PutItem(ctx context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastPutItemIn = params
	if m.putItemErr != nil {
		return nil, m.putItemErr
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDynamo) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastDeleteIn = params
	if m.deleteItemErr != nil {
		return nil, m.deleteItemErr
	}
	if m.deleteItemOut != nil {
		return m.deleteItemOut, nil
	}
	return &dynamodb.DeleteItemOutput{}, nil
}

func (m *mockDynamo) Query(ctx context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastQueryIn = params
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	if m.queryOut != nil {
		return m.queryOut, nil
	}
	return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil
}
