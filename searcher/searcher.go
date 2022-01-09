package searcher

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"github.com/y-yagi/hikimi/db"
	"github.com/y-yagi/hikimi/downloader"
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
		return downloader.Run(c.String("bucket"), []string{selectedKey}, downloadPath, session)
	}

	selectedKey = filepath.Dir(selectedKey) + "/"
	fmt.Printf("selectedKey %v\n", selectedKey)
	svc := s3.New(session)
	err = svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(c.String("bucket")),
		Prefix: aws.String(selectedKey),
	}, func(p *s3.ListObjectsOutput, last bool) (shouldContinue bool) {
		keys := []string{}
		for _, object := range p.Contents {
			keys = append(keys, *object.Key)
		}

		err = downloader.Run(c.String("bucket"), keys, downloadPath, session)
		return true
	})

	return err
}
