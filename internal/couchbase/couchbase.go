// Package couchbase provides a generic abstraction layer over the Couchbase Go SDK.
// This package simplifies common operations and provides type-safe CRUD operations
// with built-in error handling and context support.
package couchbase

import (
	"context"
	"errors"
	"fmt"

	"github.com/couchbase/gocb/v2"
)

// Couchbase is a generic wrapper around Couchbase SDK operations.
// It provides type-safe CRUD operations for any type T and handles
// common patterns like CAS (Compare-And-Swap) operations automatically.
type Couchbase[T any] struct {
	cluster    *gocb.Cluster
	bucket     *gocb.Bucket
	collection *gocb.Collection
}

// NewCouchbase creates a new generic Couchbase wrapper instance.
// All parameters are required and the function will return an error if any are nil.
func NewCouchbase[T any](cluster *gocb.Cluster, bucket *gocb.Bucket, collection *gocb.Collection) (*Couchbase[T], error) {
	if cluster == nil || bucket == nil || collection == nil {
		return nil, errors.New("invalid Couchbase parameters: cluster, bucket, and collection must not be nil")
	}

	return &Couchbase[T]{
		cluster:    cluster,
		bucket:     bucket,
		collection: collection,
	}, nil
}

// Insert creates a new document in Couchbase with the given key and value.
// Returns an error if the document already exists or if the operation fails.
func (c *Couchbase[T]) Insert(ctx context.Context, key string, value T, insertOptions *gocb.InsertOptions) error {
	if insertOptions == nil {
		insertOptions = new(gocb.InsertOptions)
	}
	insertOptions.Context = ctx

	_, err := c.collection.Insert(key, value, insertOptions)
	if err != nil {
		return fmt.Errorf("failed to insert document with key %s: %w", key, err)
	}

	return nil
}

// Get retrieves a document from Couchbase by key and unmarshals it into type T.
// Automatically sets CAS values on objects that implement CasSetter interface.
func (c *Couchbase[T]) Get(ctx context.Context, key string, opts *gocb.GetOptions) (*T, error) {
	if opts == nil {
		opts = new(gocb.GetOptions)
	}
	opts.Context = ctx

	res, err := c.collection.Get(key, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get document with key %s: %w", key, err)
	}

	var v T
	if err := res.Content(&v); err != nil {
		return nil, fmt.Errorf("failed to parse document content for key %s: %w", key, err)
	}

	if s, ok := any(&v).(CasSetter); ok {
		s.Set(uint64(res.Cas()))
	}

	return &v, nil
}

// Replace updates an existing document in Couchbase with new content.
// Automatically updates CAS values on objects that implement CasSetter interface.
func (c *Couchbase[T]) Replace(ctx context.Context, key string, v *T, opts *gocb.ReplaceOptions) error {
	if opts == nil {
		opts = new(gocb.ReplaceOptions)
	}
	opts.Context = ctx

	res, err := c.collection.Replace(key, v, opts)
	if err != nil {
		return fmt.Errorf("failed to get document with key %s: %w", key, err)
	}

	if s, ok := any(&v).(CasSetter); ok {
		s.Set(uint64(res.Cas()))
	}

	return nil
}

// Remove deletes a document from Couchbase by key.
// Does not return an error if the document doesn't exist.
func (c *Couchbase[T]) Remove(ctx context.Context, key string, opts *gocb.RemoveOptions) error {
	if opts == nil {
		opts = new(gocb.RemoveOptions)
	}
	opts.Context = ctx

	_, err := c.collection.Remove(key, opts)
	if err != nil && !errors.Is(err, gocb.ErrDocumentNotFound) {
		return fmt.Errorf("failed to remove document with key %s: %w", key, err)
	}

	return nil
}

// Query executes a N1QL query and returns the results as a slice of type T.
// Automatically marshals each row into the specified type.
func (c *Couchbase[T]) Query(ctx context.Context, query string, opts *gocb.QueryOptions) ([]T, error) {
	if opts == nil {
		opts = new(gocb.QueryOptions)
	}
	opts.Context = ctx

	result, err := c.cluster.Query(query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	var items []T
	for result.Next() {
		var item T
		if err := result.Row(&item); err != nil {
			return nil, fmt.Errorf("failed to parse query row: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// Collection returns the underlying Couchbase collection for advanced operations.
func (c *Couchbase[T]) Collection() *gocb.Collection {
	return c.collection
}

// Close closes the Couchbase cluster connection.
func (c *Couchbase[T]) Close() error {
	return c.cluster.Close(nil)
}
