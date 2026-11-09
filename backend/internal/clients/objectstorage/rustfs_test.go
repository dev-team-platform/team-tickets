package objectstorage

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/trannhanlv2004/team-tickets/internal/configs"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewUsesConfiguredEndpointBucketAndCredentials(t *testing.T) {
	var host, path, authorization string
	client, err := New(context.Background(), configs.ObjectStorageConfig{
		Endpoint:       "http://storage.test",
		BucketName:     "tickets",
		Region:         "us-east-1",
		AccessKey:      "test-access",
		SecretKey:      "test-secret",
		ForcePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	opts := client.RustFS.Options()
	opts.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		host, path, authorization = r.URL.Host, r.URL.Path, r.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}
	client.RustFS = s3.New(opts)
	_, err = client.RustFS.HeadBucket(context.Background(), &s3.HeadBucketInput{Bucket: aws.String(client.Bucket)})
	if err != nil {
		t.Fatal(err)
	}
	if host != "storage.test" || path != "/tickets" || !strings.Contains(authorization, "Credential=test-access/") {
		t.Fatal("request did not use the configured bucket and credentials")
	}
}
