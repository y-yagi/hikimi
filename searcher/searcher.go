package searcher

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"github.com/y-yagi/hikimi/db"
	"github.com/y-yagi/hikimi/downloader"
)

func Run(c *cli.Context, database, downloadPath string, cfg aws.Config) error {
	repo := db.NewRepository(database)
	if err := repo.InitDB(); err != nil {
		return fmt.Errorf("failed to create db %v", err)
	}

	musics, err := repo.Search(c.String("search"), c.String("bucket"))
	if err != nil {
		return fmt.Errorf("failed to search %v", err)
	}

	buf := ""
	stdout := new(bytes.Buffer)
	for _, music := range musics {
		buf += music.Key + "\n"
	}

	cmd := exec.Command("sh", "-c", "peco")
	cmd.Env = append(cmd.Env, "TERM=screen")
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

	if action == "directory" {
		selectedKey = filepath.Dir(selectedKey) + "/"
	}

	fmt.Printf("selectedKey %v\n", selectedKey)
	return downloader.Run(c.String("bucket"), selectedKey, downloadPath, cfg)
}
