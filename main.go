package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
)

func main() {
	os.Exit(run(os.Args))
}

func msg(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		return 1
	}
	return 0
}

func run(args []string) int {
	app := cli.NewApp()
	app.Name = "hikimi"
	app.Usage = "CLI for Wasabi"
	app.Version = "0.0.1"
	app.Action = appRun
	app.Flags = flags()

	return msg(app.Run(args))
}

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "secret",
			Aliases:  []string{"s"},
			Usage:    "secret for Wasabi",
			EnvVars:  []string{"WASABI_SECRET"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "accesskey",
			Aliases:  []string{"k"},
			Usage:    "access key for Wasabi",
			EnvVars:  []string{"WASABI_ACCESS_KEY"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "bucket",
			Aliases:  []string{"b"},
			Usage:    "bucket for Wasabi",
			EnvVars:  []string{"WASABI_BUCKET"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "prefix",
			Aliases: []string{"p"},
			Usage:   "prefix of bucket",
		},
		&cli.StringFlag{
			Name:    "region",
			Aliases: []string{"r"},
			Usage:   "region of Wasabi",
			Value:   "us-east-1",
		},
		&cli.BoolFlag{
			Name:    "generate",
			Aliases: []string{"g"},
			Usage:   "generate file list",
		},
	}
}

func appRun(c *cli.Context) error {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(c.String("accesskey"), c.String("secret"), ""),
		Endpoint:         aws.String("https://s3.wasabisys.com"),
		Region:           aws.String(c.String("region")),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession := session.New(s3Config)
	svc := s3.New(newSession)

	err := svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(c.String("bucket")),
		Prefix: aws.String(c.String("prefix")),
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		if !c.Bool("generate") {
			for _, obj := range p.Contents {
				fmt.Println(*obj.Key)
			}
		} else {
			if err := generateFileList(c.String("bucket"), p, newSession); err != nil {
				fmt.Printf("error generate list: %v", err)
			}
		}
		return true
	})

	if err != nil {
		return fmt.Errorf("error listing bucket: %v", err)
	}

	return nil
}

func generateFileList(bucket string, res *s3.ListObjectsOutput, session *session.Session) error {
	f, err := os.Create("dummy")
	if err != nil {
		return fmt.Errorf("failed to create file %v", err)
	}
	defer os.Remove("dummy")

	repo := NewRepository("hikimi.db")
	err = repo.InitDB()
	if err != nil {
		return fmt.Errorf("failed to create db%v", err)
	}

	musics := []*Music{}
	downloader := s3manager.NewDownloader(session)

	for _, object := range res.Contents {
		key := *object.Key
		if repo.Exist(key) {
			fmt.Printf("'%v' already exists\n", key)
			continue
		}

		_, err := downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			fmt.Printf("failed to download file '%v', %v\n", key, err)
			continue
		}

		m := &Music{Key: key}
		t, err := tag.ReadFrom(f)
		if err != nil {
			fmt.Printf("failed to read tag from file '%v', %v\n", key, err)
		}

		if t != nil {
			m.Title = t.Title()
			m.Album = t.Album()
			m.Artist = t.Artist()
		}

		musics = append(musics, m)
	}

	return repo.Insert(musics)
}
