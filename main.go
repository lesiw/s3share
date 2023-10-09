package main

import (
	"context"
)

func run(u *Uploader) error {
	u.Context = context.Background()

	if len(u.args()) < 2 {
		return errHelp
	}

	u.Bucket = u.getenv("S3SHARE_BUCKET")
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
