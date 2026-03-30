package mongodb

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestValidateFilter_AllowedOperators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter map[string]interface{}
	}{
		{
			name:   "simple equality",
			filter: map[string]interface{}{"name": "Alice"},
		},
		{
			name:   "$eq operator",
			filter: map[string]interface{}{"age": map[string]interface{}{"$eq": 25}},
		},
		{
			name:   "$gt and $lt",
			filter: map[string]interface{}{"age": map[string]interface{}{"$gt": 18, "$lt": 65}},
		},
		{
			name:   "$in operator",
			filter: map[string]interface{}{"status": map[string]interface{}{"$in": []interface{}{"active", "pending"}}},
		},
		{
			name: "$and with nested",
			filter: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"age": map[string]interface{}{"$gte": 18}},
					map[string]interface{}{"status": "active"},
				},
			},
		},
		{
			name: "$or operator",
			filter: map[string]interface{}{
				"$or": []interface{}{
					map[string]interface{}{"name": "Alice"},
					map[string]interface{}{"name": "Bob"},
				},
			},
		},
		{
			name:   "$exists operator",
			filter: map[string]interface{}{"email": map[string]interface{}{"$exists": true}},
		},
		{
			name:   "$not with allowed inner",
			filter: map[string]interface{}{"age": map[string]interface{}{"$not": map[string]interface{}{"$gt": 100}}},
		},
		{
			name:   "empty filter",
			filter: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateFilter(tt.filter); err != nil {
				t.Errorf("validateFilter() unexpected error: %v", err)
			}
		})
	}
}

func TestValidateFilter_DisallowedOperators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		filter map[string]interface{}
	}{
		{
			name:   "$where operator",
			filter: map[string]interface{}{"$where": "this.x > 10"},
		},
		{
			name:   "$regex operator",
			filter: map[string]interface{}{"name": map[string]interface{}{"$regex": ".*"}},
		},
		{
			name:   "$expr operator",
			filter: map[string]interface{}{"$expr": map[string]interface{}{"$gt": []interface{}{"$a", "$b"}}},
		},
		{
			name:   "$text operator",
			filter: map[string]interface{}{"$text": map[string]interface{}{"$search": "coffee"}},
		},
		{
			name: "nested disallowed in $and",
			filter: map[string]interface{}{
				"$and": []interface{}{
					map[string]interface{}{"name": map[string]interface{}{"$regex": "test"}},
				},
			},
		},
		{
			name:   "$type operator",
			filter: map[string]interface{}{"field": map[string]interface{}{"$type": "string"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilter(tt.filter)
			if err == nil {
				t.Fatal("validateFilter() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
