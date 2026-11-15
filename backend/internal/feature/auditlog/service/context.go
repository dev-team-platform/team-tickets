package service

import (
	"context"
)

type metadataKey struct{}

type Metadata struct {
	ActorID         string
	TraceID         string
	SpanID          string
	RequestID       string
	ClientActionID  string
	ClientRequestID string
}

// WithMetadata attaches request metadata without retaining the request itself.
func WithMetadata(ctx context.Context, metadata Metadata) context.Context {
	return context.WithValue(ctx, metadataKey{}, metadata)
}

// WithActorID attaches the actor selected by the service layer.
func WithActorID(ctx context.Context, actorID string) context.Context {
	metadata := MetadataFromContext(ctx)
	metadata.ActorID = actorID
	return WithMetadata(ctx, metadata)
}

func MetadataFromContext(ctx context.Context) Metadata {
	metadata, _ := ctx.Value(metadataKey{}).(Metadata)
	return metadata
}
