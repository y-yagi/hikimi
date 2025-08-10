package identifier

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func Run(bucket, localFilePath, uploadedFileKey string, cfg aws.Config) error {
	localFile, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	md5hash := md5.New()
	if _, err := io.Copy(md5hash, localFile); err != nil {
		return err
	}

	svc := s3.NewFromConfig(cfg)
	output, err := svc.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(uploadedFileKey),
	})
	if err != nil {
		return err
	}

	fileHash := fmt.Sprintf("\"%x\"", md5hash.Sum(nil))
	if string(fileHash) == *output.ETag {
		fmt.Printf("'%s' and '%s' are same\n", localFilePath, uploadedFileKey)
	} else {
		fmt.Printf("'%s(%s)' and '%s(%s)' aren't the same\n", localFilePath, fileHash, uploadedFileKey, *output.ETag)
	}
	return nil
}
