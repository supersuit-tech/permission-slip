package mongodb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const maxInsertDocuments = 100

// insertAction implements connectors.Action for mongodb.insert.
type insertAction struct {
	conn *MongoDBConnector
}

// insertParams are the parameters parsed from ActionRequest.Parameters.
type insertParams struct {
	Database   string        `json:"database"`
	Collection string        `json:"collection"`
	Documents  []interface{} `json:"documents"`
}

func (p *insertParams) validate() error {
	if p.Database == "" {
		return &connectors.ValidationError{Message: "missing required parameter: database"}
	}
	if p.Collection == "" {
		return &connectors.ValidationError{Message: "missing required parameter: collection"}
	}
	if len(p.Documents) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: documents"}
	}
	if len(p.Documents) > maxInsertDocuments {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("documents array exceeds maximum of %d items", maxInsertDocuments),
		}
	}
	return nil
}

// Execute inserts documents into a MongoDB collection.
func (a *insertAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params insertParams
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

	result, err := coll.InsertMany(ctx, params.Documents)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB insert failed: %v", err)}
	}

	return connectors.JSONResult(map[string]interface{}{
		"inserted_count": len(result.InsertedIDs),
		"inserted_ids":   result.InsertedIDs,
	})
}
