package main

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"testing"

	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type nopCloser struct {
	io.ReadSeeker
}

func (nopCloser) Close() error { mockFileClosed = true; return nil }

var mockFileClosed bool
var mockFileReadSeekCloser io.ReadSeekCloser
var mockFileData = []byte("filedata")
var mockFileDataEncoded = "M_PXf7Ma7qaZMzw6v2PyyFjL4omBIN0xN2lHGWjh7Ag"

func mockIO(t *testing.T) {
	realArgs := Args
	t.Cleanup(func() { Args = realArgs })
	Args = []string{"s3share", "somefile"}

	t.Setenv("S3SHARE_BUCKET", "somebucket")

	realSetupS3Client := setupS3Client
	t.Cleanup(func() { setupS3Client = realSetupS3Client })
	setupS3Client = func(context.Context) (*s3.Client, error) {
		return &s3.Client{}, nil
	}

	realStat := stat
	t.Cleanup(func() { stat = realStat })
	stat = func(string) (fs.FileInfo, error) {
		return nil, nil
	}

	mockFileClosed = false
	mockFileBuf := bytes.NewReader(mockFileData)
	mockFileReadSeekCloser := nopCloser{mockFileBuf}
	realFileReadSeekCloser := fileReadSeekCloser

	t.Cleanup(func() { fileReadSeekCloser = realFileReadSeekCloser })
	fileReadSeekCloser = func(string) (io.ReadSeekCloser, error) {
		return mockFileReadSeekCloser, nil
	}

	realS3HeadObject := s3HeadObject
	t.Cleanup(func() { s3HeadObject = realS3HeadObject })
	s3HeadObject = func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (
		*s3.HeadObjectOutput, error) {
		return &s3.HeadObjectOutput{}, nil
	}

	realS3PutObject := s3PutObject
	t.Cleanup(func() { s3PutObject = realS3PutObject })
	s3PutObject = func(context.Context, *s3.PutObjectInput) (
		*s3manager.UploadOutput, error) {
		return &s3manager.UploadOutput{}, nil
	}
}
