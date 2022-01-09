package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/manifoldco/promptui"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/hikimi/db"
	"github.com/y-yagi/hikimi/indexer"
)

type config struct {
	DataBase     string `toml:"database"`
	DownloadPath string `toml:"download_path"`
}

var (
	cfg config
)

func init() {
	err := configure.Load("hikimi", &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.DataBase) == 0 {
		cfg.DataBase = filepath.Join(configure.ConfigDir("hikimi"), "hikimi.db")
		configure.Save("hikimi", cfg)
	}
}

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
			Name:    "index",
			Aliases: []string{"i"},
			Usage:   "index files",
		},
		&cli.BoolFlag{
			Name:  "config",
			Usage: "edit config",
		},
		&cli.BoolFlag{
			Name:    "download",
			Aliases: []string{"d"},
			Usage:   "download files",
		},
		&cli.StringFlag{
			Name:  "search",
			Usage: "search and dowload files",
		},
	}
}

func appRun(c *cli.Context) error {
	if c.Bool("config") {
		return configure.Edit("hikimi", "vim")
	}

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(c.String("accesskey"), c.String("secret"), ""),
		Endpoint:         aws.String("https://s3.wasabisys.com"),
		Region:           aws.String(c.String("region")),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession := session.New(s3Config)
	svc := s3.New(newSession)

	if len(c.String("search")) != 0 {
		if err := search(c, newSession); err != nil {
			fmt.Printf("error in search: %v", err)
			return err
		}
		return nil
	}

	err := svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(c.String("bucket")),
		Prefix: aws.String(c.String("prefix")),
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		if c.Bool("index") {
			if err := indexer.Run(cfg.DataBase, c.String("bucket"), p, newSession); err != nil {
				fmt.Printf("error index files: %v", err)
			}
		} else if c.Bool("download") {
			if err := download(c.String("bucket"), p, newSession); err != nil {
				fmt.Printf("error download files: %v", err)
			}
		} else {
			for _, obj := range p.Contents {
				fmt.Println(*obj.Key)
			}
		}
		return true
	})

	if err != nil {
		return fmt.Errorf("error listing bucket: %v", err)
	}

	return nil
}

func download(bucket string, res *s3.ListObjectsOutput, session *session.Session) error {
	downloader := s3manager.NewDownloader(session)

	for _, object := range res.Contents {
		if err := downloadFile(bucket, *object.Key, downloader); err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(bucket, key string, downloader *s3manager.Downloader) error {
	basePath := "/tmp"
	if len(cfg.DownloadPath) != 0 {
		basePath = cfg.DownloadPath
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

	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to download file '%v', %v\n", key, err)
	}

	fmt.Printf("file was downloaded to '%v'\n", file)
	return nil
}

func search(c *cli.Context, session *session.Session) error {
	repo := db.NewRepository(cfg.DataBase)
	if err := repo.InitDB(); err != nil {
		return fmt.Errorf("failed to create db %v", err)
	}

	musics, err := repo.Search(c.String("search"))
	if err != nil {
		return fmt.Errorf("failed to search %v", err)
	}

	buf := ""
	stdout := new(bytes.Buffer)
	for _, music := range musics {
		buf += music.Key + "\n"
	}

	cmd := exec.Command("sh", "-c", "peco")
	cmd.Stdout = stdout
	cmd.Stdin = strings.NewReader(buf)
	if err = cmd.Run(); err != nil {
		return err
	}
	selectedKey := strings.TrimSuffix(stdout.String(), "\n")

	prompt := promptui.Select{
		Label: "Select download",
		Items: []string{"file", "directory"},
	}

	_, action, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			return nil
		}

		return fmt.Errorf("prompt failed %v", err)
	}

	if action == "file" {
		downloader := s3manager.NewDownloader(session)
		return downloadFile(c.String("bucket"), selectedKey, downloader)
	}

	selectedKey = filepath.Dir(selectedKey) + "/"
	fmt.Printf("selectedKey %v\n", selectedKey)
	svc := s3.New(session)
	err = svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(c.String("bucket")),
		Prefix: aws.String(selectedKey),
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		if err := download(c.String("bucket"), p, session); err != nil {
			fmt.Printf("error download files: %v", err)
			return false
		}
		return true
	})

	return err
}
