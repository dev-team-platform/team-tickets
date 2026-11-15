package objectstorage

import (
	"context"
	"io"
	"time"
)

// Operations defines the object storage operations required by application services.
type Operations interface {
	Upload(ctx context.Context, key string, content io.Reader, contentType ContentType) error
	Download(ctx context.Context, key string, destination io.Writer) error
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, lifetime time.Duration, download bool) (string, error)
	PresignUpload(ctx context.Context, key string, lifetime time.Duration) (string, error)
}
