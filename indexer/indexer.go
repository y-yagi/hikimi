package indexer

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/y-yagi/hikimi/db"
)

func Run(database, bucket string, res *s3.ListObjectsOutput, session *session.Session) error {
	repo := db.NewRepository(database)
	err := repo.InitDB()
	if err != nil {
		return fmt.Errorf("failed to create db%v", err)
	}

	musics := []*db.Music{}

	for _, object := range res.Contents {
		key := *object.Key
		if repo.Exist(key) {
			fmt.Printf("'%v' already exists\n", key)
			continue
		}

		m := &db.Music{Key: key}
		musics = append(musics, m)
	}

	return repo.Insert(musics)
}
