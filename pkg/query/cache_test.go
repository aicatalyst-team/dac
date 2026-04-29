package query

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// mockBackend implements Backend and counts Execute calls.
type mockBackend struct {
	calls  atomic.Int64
	result *QueryResult
	err    error
}

func (m *mockBackend) Execute(_ context.Context, _ string, _ string) (*QueryResult, error) {
	m.calls.Add(1)
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func newMockBackend(result *QueryResult) *mockBackend {
	return &mockBackend{result: result}
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCachedBackend(t *testing.T) {
	sampleResult := &QueryResult{
		Columns: []ColumnInfo{{Name: "id", Type: "int"}, {Name: "name", Type: "text"}},
		Rows:    [][]interface{}{{1, "alice"}, {2, "bob"}},
	}

	t.Run("cache hit", func(t *testing.T) {
		mock := newMockBackend(sampleResult)
		cb := NewCachedBackend(mock, time.Minute)

		r1, err := cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, err == nil, true)
		assertEqual(t, r1, sampleResult)

		r2, err := cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, err == nil, true)
		assertEqual(t, r2, sampleResult)

		assertEqual(t, mock.calls.Load(), int64(1))
	})

	t.Run("cache miss different query", func(t *testing.T) {
		mock := newMockBackend(sampleResult)
		cb := NewCachedBackend(mock, time.Minute)

		_, _ = cb.Execute(context.Background(), "conn1", "SELECT 1")
		_, _ = cb.Execute(context.Background(), "conn1", "SELECT 2")

		assertEqual(t, mock.calls.Load(), int64(2))
	})

	t.Run("cache miss different connection", func(t *testing.T) {
		mock := newMockBackend(sampleResult)
		cb := NewCachedBackend(mock, time.Minute)

		_, _ = cb.Execute(context.Background(), "conn1", "SELECT 1")
		_, _ = cb.Execute(context.Background(), "conn2", "SELECT 1")

		assertEqual(t, mock.calls.Load(), int64(2))
	})

	t.Run("ttl expiry", func(t *testing.T) {
		mock := newMockBackend(sampleResult)
		cb := NewCachedBackend(mock, 10*time.Millisecond)

		_, _ = cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, mock.calls.Load(), int64(1))

		time.Sleep(20 * time.Millisecond)

		_, _ = cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, mock.calls.Load(), int64(2))
	})

	t.Run("invalidate clears cache", func(t *testing.T) {
		mock := newMockBackend(sampleResult)
		cb := NewCachedBackend(mock, time.Minute)

		_, _ = cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, mock.calls.Load(), int64(1))

		cb.Invalidate()

		_, _ = cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, mock.calls.Load(), int64(2))
	})

	t.Run("errors are not cached", func(t *testing.T) {
		mock := &mockBackend{err: fmt.Errorf("connection refused")}
		cb := NewCachedBackend(mock, time.Minute)

		_, err := cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, err != nil, true)

		// Fix the backend so next call succeeds.
		mock.err = nil
		mock.result = sampleResult

		r, err := cb.Execute(context.Background(), "conn1", "SELECT 1")
		assertEqual(t, err == nil, true)
		assertEqual(t, r, sampleResult)

		// Both calls should have hit the backend.
		assertEqual(t, mock.calls.Load(), int64(2))
	})
}
