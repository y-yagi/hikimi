package indexer

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/y-yagi/hikimi/db"
)

func Run(database, prefix, bucket string, session *session.Session) error {
	repo := db.NewRepository(database)
	err := repo.InitDB()
	if err != nil {
		return fmt.Errorf("failed to create db%v", err)
	}

	svc := s3.New(session)

	err = svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}, func(list *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		musics := []*db.Music{}

		for _, object := range list.Contents {
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
			return false
		}

		return true
	})

	return err
}
