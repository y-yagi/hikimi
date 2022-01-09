package downloader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func Run(bucket string, keys []string, downloadPath string, session *session.Session) error {
	s3Downloader := s3manager.NewDownloader(session)
	for _, key := range keys {
		if err := downloadFile(bucket, key, downloadPath, s3Downloader); err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(bucket, key, downloadPath string, s3Downloader *s3manager.Downloader) error {
	basePath := "/tmp"
	if len(downloadPath) != 0 {
		basePath = downloadPath
	}
	dir, file := filepath.Split(key)
	fullepath := filepath.Join(basePath, dir)
	if err := os.MkdirAll(fullepath, os.FileMode(0755)); err != nil {
		return fmt.Errorf("failed to create dir %v", err)
	}

	file = filepath.Join(fullepath, file)
	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("failed to create file %v", err)
	}

	_, err = s3Downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download file '%v', %v\n", key, err)
	}

	fmt.Printf("file was downloaded to '%v'\n", file)
	return nil
}
