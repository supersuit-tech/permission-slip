package firestore

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type realRunner struct {
	client *firestore.Client
}

func newRealRunner(ctx context.Context, projectID string, opts []option.ClientOption) (*realRunner, error) {
	client, err := firestore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, mapFirestoreError(err)
	}
	return &realRunner{client: client}, nil
}

func (r *realRunner) close() error {
	return r.client.Close()
}

func (r *realRunner) getDocument(ctx context.Context, path string) (map[string]interface{}, error) {
	snap, err := r.client.Doc(path).Get(ctx)
	if err != nil {
		return nil, mapFirestoreError(err)
	}
	if !snap.Exists() {
		return nil, nil
	}
	return snap.Data(), nil
}

func (r *realRunner) setDocument(ctx context.Context, path string, data map[string]interface{}, merge bool) error {
	var opts []firestore.SetOption
	if merge {
		opts = append(opts, firestore.MergeAll)
	}
	_, err := r.client.Doc(path).Set(ctx, data, opts...)
	if err != nil {
		return mapFirestoreError(err)
	}
	return nil
}

func (r *realRunner) updateDocument(ctx context.Context, path string, data map[string]interface{}) error {
	var updates []firestore.Update
	for k, v := range data {
		updates = append(updates, firestore.Update{Path: k, Value: v})
	}
	if len(updates) == 0 {
		return &connectors.ValidationError{Message: "data must not be empty for update"}
	}
	_, err := r.client.Doc(path).Update(ctx, updates)
	if err != nil {
		return mapFirestoreError(err)
	}
	return nil
}

func (r *realRunner) deleteDocument(ctx context.Context, path string) error {
	_, err := r.client.Doc(path).Delete(ctx)
	if err != nil {
		return mapFirestoreError(err)
	}
	return nil
}

type queryFilter struct {
	Field string
	Op    string
	Value interface{}
}

type orderClause struct {
	Field     string
	Direction string
}

func (r *realRunner) queryCollection(ctx context.Context, collectionPath string, filters []queryFilter, order []orderClause, limit int) ([]map[string]interface{}, error) {
	colRef, err := collectionRefFromClient(r.client, collectionPath)
	if err != nil {
		return nil, err
	}
	q := colRef.Query
	for _, f := range filters {
		if f.Op != "==" {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("unsupported filter op %q (only == is supported)", f.Op)}
		}
		q = q.Where(f.Field, "==", f.Value)
	}
	for _, o := range order {
		dir := firestore.Asc
		if strings.EqualFold(o.Direction, "desc") {
			dir = firestore.Desc
		}
		q = q.OrderBy(o.Field, dir)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	iter := q.Documents(ctx)
	defer iter.Stop()
	var out []map[string]interface{}
	for {
		snap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, mapFirestoreError(err)
		}
		row := map[string]interface{}{
			"id":   snap.Ref.ID,
			"data": snap.Data(),
		}
		out = append(out, row)
	}
	return out, nil
}

func collectionRefFromClient(c *firestore.Client, path string) (*firestore.CollectionRef, error) {
	segs := splitFirestorePath(path)
	if len(segs) == 0 {
		return nil, &connectors.ValidationError{Message: "invalid collection path"}
	}
	if len(segs)%2 == 0 {
		return nil, &connectors.ValidationError{Message: "collection path must end at a collection (odd number of path segments)"}
	}
	col := c.Collection(segs[0])
	for i := 1; i < len(segs); i += 2 {
		if i+1 >= len(segs) {
			return col, nil
		}
		col = col.Doc(segs[i]).Collection(segs[i+1])
	}
	return col, nil
}
