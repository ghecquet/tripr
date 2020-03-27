package main

import (
	"context"
	"fmt"

	"github.com/coreos/etcd/etcdserver/etcdserverpb"
)

type Handler struct{}

// Range gets the keys in the range from the key-value store.
func (h *Handler) Range(context.Context, *etcdserverpb.RangeRequest) (*etcdserverpb.RangeResponse, error) {
	fmt.Println("Going here")
	return &etcdserverpb.RangeResponse{}, nil
}

// Put puts the given key into the key-value store.
// A put request increments the revision of the key-value store
// and generates one event in the event history.
func (h *Handler) Put(context.Context, *etcdserverpb.PutRequest) (*etcdserverpb.PutResponse, error) {
	return nil, nil
}

// DeleteRange deletes the given range from the key-value store.
// A delete request increments the revision of the key-value store
// and generates a delete event in the event history for every deleted key.
func (h *Handler) DeleteRange(context.Context, *etcdserverpb.DeleteRangeRequest) (*etcdserverpb.DeleteRangeResponse, error) {
	return nil, nil
}

// Txn processes multiple requests in a single transaction.
// A txn request increments the revision of the key-value store
// and generates events with the same revision for every completed request.
// It is not allowed to modify the same key several times within one txn.
func (h *Handler) Txn(context.Context, *etcdserverpb.TxnRequest) (*etcdserverpb.TxnResponse, error) {
	return nil, nil
}

// Compact compacts the event history in the etcd key-value store. The key-value
// store should be periodically compacted or the event history will continue to grow
// indefinitely.
func (h *Handler) Compact(context.Context, *etcdserverpb.CompactionRequest) (*etcdserverpb.CompactionResponse, error) {
	return nil, nil
}
