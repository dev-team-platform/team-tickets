package objectstorage

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/trannhanlv2004/team-tickets/internal/config"
)

func TestPresignedURLs(t *testing.T) {
	client, err := New(context.Background(), config.ObjectStorageConfig{
		Endpoint:       "https://storage.example",
		BucketName:     "bucket",
		Region:         "region",
		AccessKey:      "access",
		SecretKey:      "secret",
		ForcePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var storage Operations = client
	signed, err := storage.PresignGet(context.Background(), "report.xlsx", 15*time.Minute, true)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("response-content-disposition") != "attachment" || parsed.Query().Get("X-Amz-Expires") != "900" {
		t.Fatalf("unexpected URL: %s", signed)
	}
	if _, err := storage.PresignUpload(context.Background(), "report.xlsx", 8*24*time.Hour); err == nil {
		t.Fatal("expected invalid expiry")
	}
	if _, err := storage.PresignGet(context.Background(), " ", time.Minute, false); err == nil {
		t.Fatal("expected invalid key")
	}
}
