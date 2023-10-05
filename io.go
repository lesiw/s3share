package main

import (
	"context"
	"fmt"
	"io"
	"os"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	s3manager "github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

var Args = os.Args

var setupS3Client = func(ctx context.Context) (*s3.Client, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg), nil
}

var stat = os.Stat

var fileReadSeekCloser = func(path string) (io.ReadSeekCloser, error) {
	return os.Open(path)
}

var s3HeadObject = s3Client.HeadObject

var s3PutObject = func(ctx context.Context, input *s3.PutObjectInput) (
	*s3manager.UploadOutput, error) {

	return s3manager.NewUploader(s3Client).Upload(ctx, input)
}
