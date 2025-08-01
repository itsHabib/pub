package couchbase

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

// Transactions provides a wrapper around Couchbase distributed transactions.
// It simplifies transaction execution with consistent configuration and error handling.
type Transactions struct {
	cluster *gocb.Cluster
}

// NewTransactions creates a new transaction manager for the given cluster.
func NewTransactions(cluster *gocb.Cluster) (*Transactions, error) {
	if cluster == nil {
		return nil, fmt.Errorf("couchbase cluster cannot be nil")
	}

	return &Transactions{
		cluster: cluster,
	}, nil
}

// Transaction executes a function within a Couchbase distributed transaction.
// Returns the transaction ID on success or an error if the transaction fails.
func (t *Transactions) Transaction(fn TransactionAttempt) (string, error) {
	opts := gocb.TransactionOptions{
		DurabilityLevel: gocb.DurabilityLevelNone,
		Timeout:         time.Second * 10,
	}
	run := func(actx *gocb.TransactionAttemptContext) error {
		return fn(newTransactionRunner(actx))
	}

	res, err := t.cluster.Transactions().Run(run, &opts)
	if err != nil {
		return "", fmt.Errorf("failed to run transaction: %w", err)
	}

	return res.TransactionID, nil
}

// transactionRunner wraps the Couchbase transaction context to provide a simpler interface.
type transactionRunner struct {
	ctx *gocb.TransactionAttemptContext
}

// newTransactionRunner creates a new transaction runner wrapper.
func newTransactionRunner(ctx *gocb.TransactionAttemptContext) *transactionRunner {
	return &transactionRunner{ctx: ctx}
}

// Get retrieves a document within the transaction context.
func (t *transactionRunner) Get(tc TransactionCollection, key string) (*gocb.TransactionGetResult, error) {
	return t.ctx.Get(tc.Collection(), key)
}

// Insert creates a new document within the transaction context.
func (t *transactionRunner) Insert(tc TransactionCollection, key string, value any) (*gocb.TransactionGetResult, error) {
	return t.ctx.Insert(tc.Collection(), key, value)
}

// Replace updates an existing document within the transaction context.
func (t *transactionRunner) Replace(doc *gocb.TransactionGetResult, value any) (*gocb.TransactionGetResult, error) {
	return t.ctx.Replace(doc, value)
}

// TransactionRunner defines the interface for performing operations within a transaction.
type TransactionRunner interface {
	Get(tc TransactionCollection, key string) (*gocb.TransactionGetResult, error)
	Insert(tc TransactionCollection, key string, value any) (*gocb.TransactionGetResult, error)
	Replace(doc *gocb.TransactionGetResult, value any) (*gocb.TransactionGetResult, error)
}

// TransactionCollection defines the interface for collections that can participate in transactions.
type TransactionCollection interface {
	Collection() *gocb.Collection
}

// TransactionAttempt defines the signature for functions that execute within a transaction.
type TransactionAttempt func(t TransactionRunner) error
