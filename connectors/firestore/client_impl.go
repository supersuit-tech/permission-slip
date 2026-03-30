package firestore

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type realRunner struct {
	client *firestore.Client
	conn   *grpc.ClientConn // set when dialing the Firestore emulator (closed in Close)
}

func newRealRunner(ctx context.Context, projectID string, opts []option.ClientOption) (*realRunner, error) {
	client, err := firestore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, mapFirestoreError(err)
	}
	return &realRunner{client: client}, nil
}

// newRealRunnerEmulator matches cloud.google.com/go/firestore.NewClient emulator behavior
// (insecure gRPC + Bearer owner) without mutating process environment variables.
func newRealRunnerEmulator(ctx context.Context, projectID, addr string) (*realRunner, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(firestoreEmulatorCreds{}),
	)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("dialing Firestore emulator: %v", err)}
	}
	client, err := firestore.NewClient(ctx, projectID, option.WithGRPCConn(conn))
	if err != nil {
		_ = conn.Close()
		return nil, mapFirestoreError(err)
	}
	return &realRunner{client: client, conn: conn}, nil
}

// firestoreEmulatorCreds mirrors cloud.google.com/go/firestore emulatorCreds (unexported upstream).
type firestoreEmulatorCreds struct{}

func (firestoreEmulatorCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer owner"}, nil
}

func (firestoreEmulatorCreds) RequireTransportSecurity() bool { return false }

func (r *realRunner) close() error {
	var err error
	if r.client != nil {
		err = r.client.Close()
	}
	if r.conn != nil {
		if cerr := r.conn.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

func (r *realRunner) getDocument(ctx context.Context, path string) (map[string]interface{}, error) {
	snap, err := r.client.Doc(path).Get(ctx)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return nil, nil
		}
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
