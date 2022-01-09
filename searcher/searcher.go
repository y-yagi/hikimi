package searcher

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"github.com/y-yagi/hikimi/db"
)

func Run(c *cli.Context, database, downloadPath string, session *session.Session) error {
	repo := db.NewRepository(database)
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
		return downloadFile(c.String("bucket"), selectedKey, downloadPath, downloader)
	}

	selectedKey = filepath.Dir(selectedKey) + "/"
	fmt.Printf("selectedKey %v\n", selectedKey)
	svc := s3.New(session)
	err = svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(c.String("bucket")),
		Prefix: aws.String(selectedKey),
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		if err := download(c.String("bucket"), downloadPath, p, session); err != nil {
			fmt.Printf("error download files: %v", err)
			return false
		}
		return true
	})

	return err
}

func download(bucket, downloadPath string, res *s3.ListObjectsOutput, session *session.Session) error {
	downloader := s3manager.NewDownloader(session)

	for _, object := range res.Contents {
		if err := downloadFile(bucket, *object.Key, downloadPath, downloader); err != nil {
			return err
		}
	}

	return nil
}

func downloadFile(bucket, key, downloadPath string, downloader *s3manager.Downloader) error {
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
