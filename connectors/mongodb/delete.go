package mongodb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deleteAction implements connectors.Action for mongodb.delete.
type deleteAction struct {
	conn *MongoDBConnector
}

// deleteParams are the parameters parsed from ActionRequest.Parameters.
type deleteParams struct {
	Database   string                 `json:"database"`
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
	Multi      bool                   `json:"multi"`
}

func (p *deleteParams) validate() error {
	if p.Database == "" {
		return &connectors.ValidationError{Message: "missing required parameter: database"}
	}
	if p.Collection == "" {
		return &connectors.ValidationError{Message: "missing required parameter: collection"}
	}
	if len(p.Filter) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: filter (empty filter would delete all documents)"}
	}
	if err := validateFilter(p.Filter); err != nil {
		return err
	}
	return nil
}

// Execute deletes documents from a MongoDB collection.
func (a *deleteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteParams
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

	if params.Multi {
		result, err := coll.DeleteMany(ctx, filter)
		if err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB delete failed: %v", err)}
		}
		return connectors.JSONResult(map[string]interface{}{
			"deleted_count": result.DeletedCount,
		})
	}

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB delete failed: %v", err)}
	}
	return connectors.JSONResult(map[string]interface{}{
		"deleted_count": result.DeletedCount,
	})
}
