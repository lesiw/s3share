package main

import (
	"errors"
	"io"
	"io/fs"
	"testing"

	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"gotest.tools/v3/assert"
)

func TestRunNoArgs(t *testing.T) {
	r := newTestRun(t)
	r.Uploader.Args = &[]string{}

	err := run(r.Uploader)

	assert.ErrorIs(t, err, errHelp)
}

func TestRunNoEnv(t *testing.T) {
	r := newTestRun(t)
	t.Setenv("S3SHARE_BUCKET", "")

	err := run(r.Uploader)

	assert.ErrorIs(t, err, errEnvNotSet)
}

func TestRunSetupClientError(t *testing.T) {
	r := newTestRun(t)
	stopErr := errors.New("mock error")
	r.Uploader.SetupClient = func() error {
		r.SetupClientCalls++
		return stopErr
	}

	err := run(r.Uploader)

	assert.ErrorIs(t, err, stopErr)
	assert.Equal(t, r.SetupClientCalls, 1)
}

func TestRunCallsUploadFile(t *testing.T) {
	r := newTestRun(t)
	r.Uploader.Args = &[]string{"s3share", "file1", "file2", "file with spaces"}

	err := run(r.Uploader)

	assert.NilError(t, err)
	assert.DeepEqual(t, r.UploadFileCalls, []string{"file1", "file2", "file with spaces"})
}

func TestRunUploadFileError(t *testing.T) {
	r := newTestRun(t)
	errUpload := errors.New("mock error")
	r.Uploader.UploadFile = func(string) (string, error) {
		return "", errUpload
	}

	err := run(r.Uploader)

	assert.ErrorIs(t, err, errUpload)
}

func TestUploadFileStatFail(t *testing.T) {
	r := newTestRun(t)
	var statFile string
	r.Uploader.Stat = func(file string) (fs.FileInfo, error) {
		statFile = file
		return nil, errors.New("mock error")
	}

	r.Uploader.UploadFile = nil
	_, err := r.Uploader.uploadFile("somefile")

	assert.ErrorContains(t, err, "file does not exist or cannot be read")
	assert.Equal(t, statFile, "somefile")
}

func TestUploadFileOpenFail(t *testing.T) {
	r := newTestRun(t)

	var filename string
	fileErr := errors.New("mock error")
	r.Uploader.OpenFile = func(file string) (io.ReadSeekCloser, error) {
		filename = file
		return nil, fileErr
	}

	r.Uploader.UploadFile = nil
	_, err := r.Uploader.uploadFile("somefile")

	assert.Equal(t, filename, "somefile")
	assert.Equal(t, r.MockFileClosed, false)
	assert.ErrorIs(t, err, fileErr)
}

func TestUploadFileObjectExists(t *testing.T) {
	r := newTestRun(t)
	r.Uploader.ObjectExists = func(string) (bool, error) {
		return true, nil
	}

	r.Uploader.UploadFile = nil
	url, err := r.Uploader.uploadFile("somefile")

	assert.NilError(t, err)
	assert.Equal(t, r.MockFileClosed, true)
	assert.Equal(t, url, "https://somebucket.s3.amazonaws.com/"+
		"M_PXf7Ma7qaZMzw6v2PyyFjL4omBIN0xN2lHGWjh7Ag/somefile")
	assert.Equal(t, len(r.PutObjectCalls), 0)
}

func TestUploadFileObjectExistsErr(t *testing.T) {
	r := newTestRun(t)
	errObject := errors.New("mock error")
	r.Uploader.ObjectExists = func(string) (bool, error) {
		return false, errObject
	}

	r.Uploader.UploadFile = nil
	_, err := r.Uploader.uploadFile("somefile")

	assert.ErrorIs(t, err, errObject)
	assert.Equal(t, r.MockFileClosed, true)
}

func TestUploadFileObjectDoesNotExist(t *testing.T) {
	r := newTestRun(t)
	r.Uploader.ObjectExists = func(string) (bool, error) {
		return false, nil
	}

	r.Uploader.UploadFile = nil
	url, err := r.Uploader.uploadFile("somefile")

	assert.NilError(t, err)
	assert.Equal(t, r.MockFileClosed, true)
	assert.Equal(t, url, "https://somebucket.s3.amazonaws.com/"+
		"M_PXf7Ma7qaZMzw6v2PyyFjL4omBIN0xN2lHGWjh7Ag/somefile")
	assert.Equal(t, len(r.PutObjectCalls), 1)
	assert.Equal(t, *r.PutObjectCalls[0].Bucket, "somebucket")
	assert.Equal(t, *r.PutObjectCalls[0].Key, mockFileDataEncoded+"/somefile")
	assert.Equal(t, r.PutObjectCalls[0].ACL, s3types.ObjectCannedACLPublicRead)
}

func TestUploadFileObjectPutError(t *testing.T) {
	r := newTestRun(t)
	putErr := errors.New("mock error")
	r.Uploader.ObjectExists = func(string) (bool, error) {
		return false, nil
	}
	r.Uploader.PutObject = func(in *s3.PutObjectInput) (*s3manager.UploadOutput, error) {
		r.PutObjectCalls = append(r.PutObjectCalls, in)
		return nil, putErr
	}

	r.Uploader.UploadFile = nil
	_, err := r.Uploader.uploadFile("somefile")

	assert.ErrorIs(t, err, putErr)
	assert.Equal(t, r.MockFileClosed, true)
	assert.Equal(t, len(r.PutObjectCalls), 1)
	assert.Equal(t, *r.PutObjectCalls[0].Bucket, "somebucket")
	assert.Equal(t, *r.PutObjectCalls[0].Key, mockFileDataEncoded+"/somefile")
	assert.Equal(t, r.PutObjectCalls[0].ACL, s3types.ObjectCannedACLPublicRead)
}
func TestObjectExistsHeadSuccess(t *testing.T) {
	r := newTestRun(t)

	var bucket, key string
	r.Uploader.HeadObject = func(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
		bucket = *input.Bucket
		key = *input.Key
		return nil, nil
	}

	r.Uploader.ObjectExists = nil
	exists, err := r.Uploader.objectExists("some/key")

	assert.NilError(t, err)
	assert.Equal(t, bucket, r.Uploader.Bucket)
	assert.Equal(t, key, "some/key")
	assert.Equal(t, exists, true)
}

func TestObjectExistsHeadFailure(t *testing.T) {
	r := newTestRun(t)

	var bucket, key string
	r.Uploader.HeadObject = func(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
		bucket = *input.Bucket
		key = *input.Key
		return nil, &smithy.GenericAPIError{Code: "NotFound"}
	}

	r.Uploader.ObjectExists = nil
	exists, err := r.Uploader.objectExists("some/key")

	assert.NilError(t, err)
	assert.Equal(t, bucket, r.Uploader.Bucket)
	assert.Equal(t, key, "some/key")
	assert.Equal(t, exists, false)
}

func TestObjectExistsUnexpectedError(t *testing.T) {
	r := newTestRun(t)

	headErr := errors.New("mock error")
	var bucket, key string
	r.Uploader.HeadObject = func(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
		bucket = *input.Bucket
		key = *input.Key
		return nil, headErr
	}

	r.Uploader.ObjectExists = nil
	_, err := r.Uploader.objectExists("some/key")

	assert.ErrorIs(t, err, headErr)
	assert.Equal(t, bucket, r.Uploader.Bucket)
	assert.Equal(t, key, "some/key")
}
