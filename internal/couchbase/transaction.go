package couchbase

import (
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

type Transactions struct {
	cluster *gocb.Cluster
}

func NewTransactions(cluster *gocb.Cluster) (*Transactions, error) {
	if cluster == nil {
		return nil, fmt.Errorf("couchbase cluster cannot be nil")
	}

	return &Transactions{
		cluster: cluster,
	}, nil
}

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

type transactionRunner struct {
	ctx *gocb.TransactionAttemptContext
}

func newTransactionRunner(ctx *gocb.TransactionAttemptContext) *transactionRunner {
	return &transactionRunner{ctx: ctx}
}

func (t *transactionRunner) Get(tc TransactionCollection, key string) (*gocb.TransactionGetResult, error) {
	return t.ctx.Get(tc.Collection(), key)
}

func (t *transactionRunner) Insert(tc TransactionCollection, key string, value any) (*gocb.TransactionGetResult, error) {
	return t.ctx.Insert(tc.Collection(), key, value)
}

func (t *transactionRunner) Replace(doc *gocb.TransactionGetResult, value any) (*gocb.TransactionGetResult, error) {
	return t.ctx.Replace(doc, value)
}

type TransactionRunner interface {
	Get(tc TransactionCollection, key string) (*gocb.TransactionGetResult, error)
	Insert(tc TransactionCollection, key string, value any) (*gocb.TransactionGetResult, error)
	Replace(doc *gocb.TransactionGetResult, value any) (*gocb.TransactionGetResult, error)
}

type TransactionCollection interface {
	Collection() *gocb.Collection
}

type TransactionAttempt func(t TransactionRunner) error
