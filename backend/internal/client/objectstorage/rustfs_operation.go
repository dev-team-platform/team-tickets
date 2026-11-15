package objectstorage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var _ Operations = (*Client)(nil)

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("object key is required")
	}
	return nil
}

func (c *Client) Upload(ctx context.Context, key string, content io.Reader, contentType ContentType) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if content == nil || strings.TrimSpace(string(contentType)) == "" {
		return fmt.Errorf("content and content type are required")
	}
	_, err := c.RustFS.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.Bucket),
		Key:         aws.String(key),
		Body:        content,
		ContentType: aws.String(string(contentType)),
	})
	return err
}

func (c *Client) Download(ctx context.Context, key string, destination io.Writer) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if destination == nil {
		return fmt.Errorf("destination is required")
	}
	output, err := c.RustFS.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer output.Body.Close()
	_, err = io.Copy(destination, output.Body)
	return err
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	_, err := c.RustFS.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	})
	return err
}

// PresignGet returns a temporary URL for viewing or downloading an object.
func (c *Client) PresignGet(ctx context.Context, key string, lifetime time.Duration, download bool) (string, error) {
	if err := validatePresign(key, lifetime); err != nil {
		return "", err
	}
	disposition := "inline"
	if download {
		disposition = "attachment"
	}
	result, err := s3.NewPresignClient(c.RustFS).PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(c.Bucket),
		Key:                        aws.String(key),
		ResponseContentDisposition: aws.String(disposition),
	}, s3.WithPresignExpires(lifetime))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func (c *Client) PresignUpload(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	if err := validatePresign(key, lifetime); err != nil {
		return "", err
	}
	result, err := s3.NewPresignClient(c.RustFS).PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(lifetime))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func validatePresign(key string, lifetime time.Duration) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if lifetime < time.Second || lifetime > 7*24*time.Hour {
		return fmt.Errorf("URL lifetime must be between one second and seven days")
	}
	return nil
}
