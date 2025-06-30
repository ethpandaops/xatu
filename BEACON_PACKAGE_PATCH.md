# Required Patch for ethpandaops/beacon Package

To enable data column sidecar events in xatu, the following changes need to be made to `github.com/ethpandaops/beacon`:

## 1. Add OnDataColumnSidecar method to Node interface

**File:** `pkg/beacon/beacon.go`

Add this method to the `Node` interface after `OnBlobSidecar`:

```go
// OnDataColumnSidecar is called when a data column sidecar is received.
OnDataColumnSidecar(ctx context.Context, handler func(ctx context.Context, ev *DataColumnSidecarEvent) error)
```

## 2. Implement OnDataColumnSidecar method

**File:** `pkg/beacon/subscriber.go`

Add this implementation after the `OnBlobSidecar` method:

```go
func (n *node) OnDataColumnSidecar(ctx context.Context, handler func(ctx context.Context, event *DataColumnSidecarEvent) error) {
	n.broker.On(topicDataColumnSidecar, func(event *DataColumnSidecarEvent) {
		n.handleSubscriberError(handler(ctx, event), topicDataColumnSidecar)
	})
}
```

## Summary

The data column sidecar infrastructure already exists in the beacon package:
- ✅ `DataColumnSidecarEvent` struct is defined
- ✅ `topicDataColumnSidecar` constant exists  
- ✅ `handleDataColumnSidecar` method exists
- ✅ `publishDataColumnSidecar` method exists
- ❌ `OnDataColumnSidecar` public method is missing (this patch adds it)

Once this patch is applied to the beacon package and released, the commented code in `/Users/samcm/go/src/github.com/ethpandaops/xatu/pkg/sentry/sentry.go` can be uncommented to enable full data column sidecar support.