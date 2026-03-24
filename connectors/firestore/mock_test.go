package firestore

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type mockRunner struct {
	mu sync.Mutex

	getData map[string]map[string]interface{}
	getErr  error

	setErr error

	updateErr error

	deleteErr error

	queryDocs [][]map[string]interface{}
	queryErr  error
}

func newMockRunner() *mockRunner {
	return &mockRunner{getData: make(map[string]map[string]interface{})}
}

func (m *mockRunner) getDocument(ctx context.Context, path string) (map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		if st, ok := status.FromError(m.getErr); ok && st.Code() == codes.NotFound {
			return nil, nil
		}
		return nil, m.getErr
	}
	d, ok := m.getData[path]
	if !ok {
		return nil, nil
	}
	out := make(map[string]interface{}, len(d))
	for k, v := range d {
		out[k] = v
	}
	return out, nil
}

func (m *mockRunner) setDocument(ctx context.Context, path string, data map[string]interface{}, merge bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	if merge {
		existing := m.getData[path]
		if existing == nil {
			existing = make(map[string]interface{})
		}
		for k, v := range data {
			existing[k] = v
		}
		m.getData[path] = existing
		return nil
	}
	cp := make(map[string]interface{}, len(data))
	for k, v := range data {
		cp[k] = v
	}
	m.getData[path] = cp
	return nil
}

func (m *mockRunner) updateDocument(ctx context.Context, path string, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	existing := m.getData[path]
	if existing == nil {
		existing = make(map[string]interface{})
	}
	for k, v := range data {
		existing[k] = v
	}
	m.getData[path] = existing
	return nil
}

func (m *mockRunner) deleteDocument(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.getData, path)
	return nil
}

func (m *mockRunner) queryCollection(ctx context.Context, collectionPath string, filters []queryFilter, order []orderClause, limit int) ([]map[string]interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	if len(m.queryDocs) == 0 {
		return []map[string]interface{}{}, nil
	}
	batch := m.queryDocs[0]
	m.queryDocs = m.queryDocs[1:]
	return batch, nil
}

func (m *mockRunner) close() error { return nil }

func (m *mockRunner) pushQueryBatch(docs []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queryDocs = append(m.queryDocs, docs)
}

func validServiceAccountJSON() string {
	return `{
		"type": "service_account",
		"project_id": "test-proj",
		"private_key_id": "x",
		"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy0AHB7MhgwKVqH7cncWef\n-----END RSA PRIVATE KEY-----\n",
		"client_email": "svc@test-proj.iam.gserviceaccount.com",
		"client_id": "1",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token"
	}`
}

func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"service_account_json": validServiceAccountJSON(),
	})
}
