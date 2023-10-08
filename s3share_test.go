package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"testing"

	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"gotest.tools/v3/assert"
)

func TestRunNoArgs(t *testing.T) {
	mockIO(t)

	Args = []string{}
	err := run()
	assert.ErrorIs(t, err, errHelp)
}

func TestRunNoEnv(t *testing.T) {
	mockIO(t)

	t.Setenv("S3SHARE_BUCKET", "")
	err := run()
	assert.ErrorIs(t, err, errEnvNotSet)
}

func TestSetupS3Called(t *testing.T) {
	mockIO(t)

	stopErr := errors.New("mock error")
	var setupS3ClientCalled bool
	setupS3Client = func(ctx context.Context) (*s3.Client, error) {
		setupS3ClientCalled = true
		return &s3.Client{}, stopErr
	}

	err := run()

	assert.ErrorIs(t, err, stopErr)
	assert.Equal(t, setupS3ClientCalled, true)
}

func TestRunStatFail(t *testing.T) {
	mockIO(t)

	var statFile string
	stat = func(file string) (fs.FileInfo, error) {
		statFile = file
		return nil, errors.New("mock error")
	}

	err := run()

	assert.ErrorContains(t, err, "file does not exist or cannot be read")
	assert.Equal(t, statFile, "somefile")
}

func TestFileReadSeekCloserFail(t *testing.T) {
	mockIO(t)

	var passedFile string
	fileErr := errors.New("mock error")
	fileReadSeekCloser = func(file string) (io.ReadSeekCloser, error) {
		passedFile = file
		return nil, fileErr
	}

	err := run()

	assert.Equal(t, passedFile, "somefile")
	assert.ErrorIs(t, err, fileErr)
}

func TestKeyExists(t *testing.T) {
	mockIO(t)

	var bucket, key string
	s3HeadObject = func(_ context.Context, input *s3.HeadObjectInput, _ ...func(*s3.Options)) (
		*s3.HeadObjectOutput, error) {
		bucket = *input.Bucket
		key = *input.Key
		return nil, nil
	}

	var calledS3PutObject bool
	s3PutObject = func(context.Context, *s3.PutObjectInput) (
		*s3manager.UploadOutput, error) {
		calledS3PutObject = true
		return nil, nil
	}

	url, err := uploadFileToBucket(context.TODO(), "somefile", "somebucket")

	assert.NilError(t, err)
	assert.Equal(t, url, "https://somebucket.s3.amazonaws.com/"+
		"M_PXf7Ma7qaZMzw6v2PyyFjL4omBIN0xN2lHGWjh7Ag/somefile")
	assert.Equal(t, bucket, "somebucket")
	assert.Equal(t, key, mockFileDataEncoded+"/somefile")
	assert.Equal(t, calledS3PutObject, false)
}

func TestKeyDoesNotExist(t *testing.T) {
	mockIO(t)

	var bucket, key string
	s3HeadObject = func(_ context.Context, input *s3.HeadObjectInput, _ ...func(*s3.Options)) (
		*s3.HeadObjectOutput, error) {
		bucket = *input.Bucket
		key = *input.Key
		return nil, &smithy.GenericAPIError{Code: "NotFound"}
	}

	var calledS3PutObject bool
	s3PutObject = func(context.Context, *s3.PutObjectInput) (
		*s3manager.UploadOutput, error) {
		calledS3PutObject = true
		return nil, nil
	}

	url, err := uploadFileToBucket(context.TODO(), "somefile", "somebucket")

	assert.NilError(t, err)
	assert.Equal(t, url, "https://somebucket.s3.amazonaws.com/"+
		"M_PXf7Ma7qaZMzw6v2PyyFjL4omBIN0xN2lHGWjh7Ag/somefile")
	assert.Equal(t, bucket, "somebucket")
	assert.Equal(t, key, mockFileDataEncoded+"/somefile")
	assert.Equal(t, calledS3PutObject, true)
	assert.Equal(t, mockFileClosed, true)
}

func TestHeadObjectFailure(t *testing.T) {
	mockIO(t)

	headErr := errors.New("mock error")
	var bucket, key string
	s3HeadObject = func(_ context.Context, input *s3.HeadObjectInput, _ ...func(*s3.Options)) (
		*s3.HeadObjectOutput, error) {
		bucket = *input.Bucket
		key = *input.Key
		return nil, headErr
	}

	var calledS3PutObject bool
	s3PutObject = func(context.Context, *s3.PutObjectInput) (
		*s3manager.UploadOutput, error) {
		calledS3PutObject = true
		return nil, nil
	}

	_, err := uploadFileToBucket(context.TODO(), "somefile", "somebucket")

	assert.ErrorIs(t, err, headErr)
	assert.Equal(t, bucket, "somebucket")
	assert.Equal(t, key, mockFileDataEncoded+"/somefile")
	assert.Equal(t, calledS3PutObject, false)
	assert.Equal(t, mockFileClosed, true)
}

func TestSuccess(t *testing.T) {
	mockIO(t)

	err := run()

	assert.NilError(t, err)
	assert.Equal(t, mockFileClosed, true)
}
