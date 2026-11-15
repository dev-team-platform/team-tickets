package service

import (
	"context"
	"testing"
)

func TestMetadataAndActorContext(t *testing.T) {
	ctx := WithMetadata(context.Background(), Metadata{
		TraceID:   "trace",
		RequestID: "request",
	})
	ctx = WithActorID(ctx, "actor")
	got := MetadataFromContext(ctx)
	if got.ActorID != "actor" || got.TraceID != "trace" || got.RequestID != "request" {
		t.Fatalf("metadata lost: %+v", got)
	}
}
