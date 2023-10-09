package main

import (
	"fmt"
	"io"
	"os"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	if err := run(new(Uploader)); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func (u *Uploader) getenv(name string) string {
	if u.Getenv != nil {
		return u.Getenv(name)
	}

	return os.Getenv(name)
}

func (u *Uploader) args() []string {
	if u.Args != nil {
		return *u.Args
	}

	return os.Args
}

func (u *Uploader) println(args ...any) (int, error) {
	if u.Println != nil {
		return u.Println(args...)
	}

	return fmt.Println(args...)
}

func (u *Uploader) stat(name string) (os.FileInfo, error) {
	if u.Stat != nil {
		return u.Stat(name)
	}

	return os.Stat(name)
}

func (u *Uploader) setupClient() error {
	if u.SetupClient != nil {
		return u.SetupClient()
	}

	cfg, err := awscfg.LoadDefaultConfig(u.Context)
	if err != nil {
		return err
	}
	u.Client = s3.NewFromConfig(cfg)
	return nil
}

func (u *Uploader) headObject(in *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	if u.HeadObject != nil {
		return u.HeadObject(in)
	}

	if u.Client == nil {
		if err := u.setupClient(); err != nil {
			return nil, err
		}
	}
	return u.Client.HeadObject(u.Context, in)
}

func (u *Uploader) openFile(path string) (io.ReadSeekCloser, error) {
	if u.OpenFile != nil {
		return u.OpenFile(path)
	}

	return os.Open(path)
}

func (u *Uploader) putObject(in *s3.PutObjectInput) (*s3manager.UploadOutput, error) {
	if u.PutObject != nil {
		return u.PutObject(in)
	}

	if u.Client == nil {
		if err := u.setupClient(); err != nil {
			return nil, err
		}
	}

	return s3manager.NewUploader(u.Client).Upload(u.Context, in)
}
