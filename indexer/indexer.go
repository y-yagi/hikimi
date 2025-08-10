package indexer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/y-yagi/hikimi/db"
)

func Run(database, prefix, bucket string, cfg aws.Config) error {
	repo := db.NewRepository(database)
	err := repo.InitDB()
	if err != nil {
		return fmt.Errorf("failed to create db%v", err)
	}

	svc := s3.NewFromConfig(cfg)

	paginator := s3.NewListObjectsV2Paginator(svc, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return err
		}

		musics := []*db.Music{}

		for _, object := range page.Contents {
			key := *object.Key
			if repo.Exist(key) {
				fmt.Printf("'%v' already exists\n", key)
				continue
			}

			m := &db.Music{Key: key, Bucket: bucket}
			musics = append(musics, m)
		}

		if err := repo.Insert(musics); err != nil {
			fmt.Printf("Insert failed %v\n", err)
			return err
		}
	}

	return nil
}
