package mongodb

import (
	"context"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateAction implements connectors.Action for mongodb.update.
type updateAction struct {
	conn *MongoDBConnector
}

// updateParams are the parameters parsed from ActionRequest.Parameters.
type updateParams struct {
	Database   string                 `json:"database"`
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
	Update     map[string]interface{} `json:"update"`
	Multi      bool                   `json:"multi"`
}

func (p *updateParams) validate() error {
	if p.Database == "" {
		return &connectors.ValidationError{Message: "missing required parameter: database"}
	}
	if p.Collection == "" {
		return &connectors.ValidationError{Message: "missing required parameter: collection"}
	}
	if len(p.Filter) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: filter"}
	}
	if len(p.Update) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: update"}
	}
	if err := validateFilter(p.Filter); err != nil {
		return err
	}
	// Update document must use update operators (keys starting with $).
	hasOperator := false
	for key := range p.Update {
		if len(key) > 0 && key[0] == '$' {
			hasOperator = true
			break
		}
	}
	if !hasOperator {
		return &connectors.ValidationError{
			Message: "update must use update operators (e.g., $set, $inc); raw document replacement is not allowed",
		}
	}
	return nil
}

// Execute updates documents in a MongoDB collection.
func (a *updateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateParams
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
	filter := mapToBsonD(params.Filter)
	update := bson.D{}
	for k, v := range params.Update {
		update = append(update, bson.E{Key: k, Value: v})
	}

	if params.Multi {
		result, err := coll.UpdateMany(ctx, filter, update)
		if err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB update failed: %v", err)}
		}
		return connectors.JSONResult(map[string]interface{}{
			"matched_count":  result.MatchedCount,
			"modified_count": result.ModifiedCount,
		})
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB update failed: %v", err)}
	}
	return connectors.JSONResult(map[string]interface{}{
		"matched_count":  result.MatchedCount,
		"modified_count": result.ModifiedCount,
	})
}
