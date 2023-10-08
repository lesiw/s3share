package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var s3Client *s3.Client
var errHelp = errors.New(`s3share [file]

Uploads files to an S3 bucket specified in the environment variable S3SHARE_BUCKET.`)
var errEnvNotSet = errors.New("S3SHARE_BUCKET environment variable not set.")

func run() (err error) {
	ctx := context.Background()

	if len(Args) < 2 {
		return errHelp
	}
	bucket := os.Getenv("S3SHARE_BUCKET")
	if bucket == "" {
		return errEnvNotSet
	}

	s3Client, err = setupS3Client(ctx)
	if err != nil {
		return err
	}

	for _, f := range Args[1:] {
		url, err := uploadFileToBucket(ctx, f, bucket)
		if err != nil {
			return err
		}
		fmt.Println(url)
	}
	return nil
}

func uploadFileToBucket(ctx context.Context, path string, bucket string) (string, error) {
	if _, err := stat(path); err != nil {
		return "", fmt.Errorf("file does not exist or cannot be read: %s", path)
	}

	file, err := fileReadSeekCloser(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", fmt.Errorf("error computing file hash: %w", err)
	}

	key := base64.RawURLEncoding.EncodeToString(sum.Sum(nil)) + "/" + filepath.Base(path)
	if ok, err := s3KeyExists(ctx, bucket, key); err != nil {
		return "", err
	} else if ok {
		return objectUrl(bucket, key), nil
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	s3PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   file,
		ACL:    s3types.ObjectCannedACLPublicRead,
	})

	return objectUrl(bucket, key), nil
}

func objectUrl(bucket string, key string) string {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)
}

func s3KeyExists(ctx context.Context, bucket string, key string) (bool, error) {
	_, err := s3HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}
