package mongodb

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// findAction implements connectors.Action for mongodb.find.
type findAction struct {
	conn *MongoDBConnector
}

// findParams are the parameters parsed from ActionRequest.Parameters.
type findParams struct {
	Database   string                 `json:"database"`
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
	Projection map[string]interface{} `json:"projection"`
	Sort       map[string]interface{} `json:"sort"`
	Limit      *int64                 `json:"limit"`
	Skip       *int64                 `json:"skip"`
}

func (p *findParams) validate() error {
	if p.Database == "" {
		return &connectors.ValidationError{Message: "missing required parameter: database"}
	}
	if p.Collection == "" {
		return &connectors.ValidationError{Message: "missing required parameter: collection"}
	}
	if p.Filter != nil {
		if err := validateFilter(p.Filter); err != nil {
			return err
		}
	}
	if p.Limit != nil && (*p.Limit < 1 || *p.Limit > maxResultLimit) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 1 and %d", maxResultLimit),
		}
	}
	if p.Skip != nil && *p.Skip < 0 {
		return &connectors.ValidationError{Message: "skip must be non-negative"}
	}
	return nil
}

// Execute queries documents from a MongoDB collection.
func (a *findAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params findParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	client, err := a.conn.connect(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(ctx)

	coll := client.Database(params.Database).Collection(params.Collection)

	filter := bson.D{}
	if params.Filter != nil {
		filter = mapToBsonD(params.Filter)
	}

	opts := options.Find()
	limit := int64(defaultResultLimit)
	if params.Limit != nil {
		limit = *params.Limit
	}
	opts.SetLimit(limit)

	if params.Skip != nil {
		opts.SetSkip(*params.Skip)
	}
	if params.Projection != nil {
		opts.SetProjection(mapToBsonD(params.Projection))
	}
	if params.Sort != nil {
		opts.SetSort(mapToBsonD(params.Sort))
	}

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB find failed: %v", err)}
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB cursor read failed: %v", err)}
	}

	if results == nil {
		results = []bson.M{}
	}

	return connectors.JSONResult(map[string]interface{}{
		"documents": results,
		"count":     len(results),
	})
}

// mapToBsonD converts a map[string]interface{} to bson.D preserving key order
// as much as Go maps allow. For most filter/sort/projection use cases this is fine.
func mapToBsonD(m map[string]interface{}) bson.D {
	d := make(bson.D, 0, len(m))
	for k, v := range m {
		d = append(d, bson.E{Key: k, Value: v})
	}
	return d
}
