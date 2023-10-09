package main

import (
	"context"
	"os"
)

func run(u *Uploader) error {
	u.Context = context.Background()

	if len(u.args()) < 2 {
		return errHelp
	}

	u.Bucket = os.Getenv("S3SHARE_BUCKET")
	if u.Bucket == "" {
		return errEnvNotSet
	}

	if err := u.setupClient(); err != nil {
		return err
	}

	for _, f := range u.args()[1:] {
		url, err := u.uploadFile(f)
		if err != nil {
			return err
		}
		u.println(url)
	}
	return nil
}
