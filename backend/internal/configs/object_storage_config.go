package configs

import (
	"fmt"
	"net/url"
	"os"
)

type ObjectStorageConfig struct {
	Endpoint       string
	BucketName     string
	Region         string
	AccessKey      string
	SecretKey      string
	ForcePathStyle bool
}

func loadObjectStorageConfig() (ObjectStorageConfig, error) {
	forcePathStyle, err := boolEnv("OBJECT_STORAGE_FORCE_PATH_STYLE", false)
	if err != nil {
		return ObjectStorageConfig{}, err
	}
	if os.Getenv("OBJECT_STORAGE_REGION") == "" || os.Getenv("OBJECT_STORAGE_BUCKET_NAME") == "" {
		return ObjectStorageConfig{}, fmt.Errorf("OBJECT_STORAGE_REGION and OBJECT_STORAGE_BUCKET_NAME are required")
	}
	if (os.Getenv("OBJECT_STORAGE_ACCESS_KEY") == "") != (os.Getenv("OBJECT_STORAGE_SECRET_KEY") == "") {
		return ObjectStorageConfig{}, fmt.Errorf("OBJECT_STORAGE_ACCESS_KEY and OBJECT_STORAGE_SECRET_KEY must be provided together")
	}
	if endpoint := os.Getenv("OBJECT_STORAGE_ENDPOINT"); endpoint != "" {
		u, err := url.Parse(endpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return ObjectStorageConfig{}, fmt.Errorf("OBJECT_STORAGE_ENDPOINT must be a valid HTTP or HTTPS URL")
		}
	}
	return ObjectStorageConfig{
		Endpoint:       os.Getenv("OBJECT_STORAGE_ENDPOINT"),
		BucketName:     os.Getenv("OBJECT_STORAGE_BUCKET_NAME"),
		Region:         os.Getenv("OBJECT_STORAGE_REGION"),
		AccessKey:      os.Getenv("OBJECT_STORAGE_ACCESS_KEY"),
		SecretKey:      os.Getenv("OBJECT_STORAGE_SECRET_KEY"),
		ForcePathStyle: forcePathStyle,
	}, nil
}
