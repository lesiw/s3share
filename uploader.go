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

	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var errHelp = errors.New(`s3share [file]

Uploads files to an S3 bucket specified in the environment variable S3SHARE_BUCKET.`)
var errEnvNotSet = errors.New("S3SHARE_BUCKET environment variable not set.")

type Uploader struct {
	// Variables.
	Args    *[]string
	Bucket  string
	Client  *S3Client
	Context context.Context

	// IO functions.
	Getenv     func(string) string
	HeadObject func(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	OpenFile   func(string) (io.ReadSeekCloser, error)
	Println    func(...any) (int, error)
	PutObject  func(*s3.PutObjectInput) (*s3manager.UploadOutput, error)
	Stat       func(string) (os.FileInfo, error)

	// Internal functions.
	ObjectExists func(string) (bool, error)
	SetupClient  func() error
	UploadFile   func(string) (string, error)
}

type S3Client struct {
	*s3.Client
}

func (u *Uploader) uploadFile(path string) (string, error) {
	if u.UploadFile != nil {
		return u.UploadFile(path)
	}

	if _, err := u.stat(path); err != nil {
		return "", fmt.Errorf("file does not exist or cannot be read: %s", path)
	}

	file, err := u.openFile(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", fmt.Errorf("error computing file hash: %w", err)
	}

	key := base64.RawURLEncoding.EncodeToString(sum.Sum(nil)) + "/" + filepath.Base(path)
	if ok, err := u.objectExists(key); err != nil {
		return "", err
	} else if ok {
		return objectUrl(u.Bucket, key), nil
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	_, err = u.putObject(&s3.PutObjectInput{
		Bucket: &u.Bucket,
		Key:    &key,
		Body:   file,
		ACL:    s3types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", err
	}

	return objectUrl(u.Bucket, key), nil
}

func objectUrl(bucket string, key string) string {
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)
}

func (u *Uploader) objectExists(key string) (bool, error) {
	if u.ObjectExists != nil {
		return u.ObjectExists(key)
	}

	_, err := u.headObject(&s3.HeadObjectInput{
		Bucket: &u.Bucket,
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

func (u *Uploader) Clone() *Uploader {
	cl := &Uploader{
		Bucket:  u.Bucket,
		Client:  u.Client,
		Context: u.Context,

		HeadObject: u.HeadObject,
		OpenFile:   u.OpenFile,
		Println:    u.Println,
		PutObject:  u.PutObject,
		Stat:       u.Stat,

		ObjectExists: u.ObjectExists,
		SetupClient:  u.SetupClient,
		UploadFile:   u.UploadFile,
	}
	if u.Args != nil {
		clargs := append([]string{}, *u.Args...)
		cl.Args = &clargs
	}
	return cl
}
