package downloader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func Run(bucket, prefix string, downloadPath string, cfg aws.Config) error {
	svc := s3.NewFromConfig(cfg)
	s3Downloader := manager.NewDownloader(svc)

	paginator := s3.NewListObjectsV2Paginator(svc, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}

		if len(page.Contents) == 0 {
			fmt.Println("Contents couldn't find. Maybe the bucket is wrong?")
			continue
		}

		for _, object := range page.Contents {
			if err := downloadFile(bucket, *object.Key, downloadPath, s3Downloader); err != nil {
				fmt.Printf("Download error: %v\n", err)
			}
		}
	}

	return nil
}

func downloadFile(bucket, key, downloadPath string, s3Downloader *manager.Downloader) error {
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
	defer f.Close()

	_, err = s3Downloader.Download(context.TODO(), f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download file '%v', %v\n", key, err)
	}

	fmt.Printf("file was downloaded to '%v'\n", file)
	return nil
}
