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
	CloseHook func()
}

func (c nopCloser) Close() error { c.CloseHook(); return nil }

var (
	mockFileData        = []byte("filedata")
	mockFileDataEncoded = "M_PXf7Ma7qaZMzw6v2PyyFjL4omBIN0xN2lHGWjh7Ag"
)

type testRun struct {
	Uploader *Uploader

	MockFile       io.ReadSeekCloser
	MockFileClosed bool

	PutObjectCalls []*s3.PutObjectInput

	ObjectExistsCalls []string
	SetupClientCalls  int
	UploadFileCalls   []string
}

var testUploader = &Uploader{
	Args:    &[]string{"s3share", "somefile"},
	Bucket:  "somebucket",
	Context: context.Background(),

	HeadObject: func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
		return &s3.HeadObjectOutput{}, nil
	},
	Stat: func(string) (fs.FileInfo, error) {
		return nil, nil
	},
}

func newTestRun(t *testing.T) *testRun {
	u := testUploader.Clone()
	run := &testRun{Uploader: u}

	buf := bytes.NewReader(mockFileData)
	mockFile := nopCloser{buf, nil}
	mockFile.CloseHook = func() {
		run.MockFileClosed = true
	}
	run.MockFile = &mockFile

	u.OpenFile = func(string) (io.ReadSeekCloser, error) {
		return run.MockFile, nil
	}
	u.PutObject = func(in *s3.PutObjectInput) (*s3manager.UploadOutput, error) {
		run.PutObjectCalls = append(run.PutObjectCalls, in)
		return &s3manager.UploadOutput{}, nil
	}

	u.ObjectExists = func(key string) (bool, error) {
		run.ObjectExistsCalls = append(run.ObjectExistsCalls, key)
		return false, nil
	}
	u.SetupClient = func() error {
		run.SetupClientCalls++
		return nil
	}
	u.UploadFile = func(file string) (string, error) {
		run.UploadFileCalls = append(run.UploadFileCalls, file)
		return "", nil
	}

	t.Setenv("S3SHARE_BUCKET", "somebucket")

	return run
}
