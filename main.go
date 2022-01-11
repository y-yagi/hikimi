package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/hikimi/indexer"
	"github.com/y-yagi/hikimi/searcher"
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
	newSession, err := session.NewSession(s3Config)
	if err != nil {
		return err
	}

	if len(c.String("search")) != 0 {
		if err := searcher.Run(c, cfg.DataBase, cfg.DownloadPath, newSession); err != nil {
			fmt.Printf("error in search: %v", err)
			return err
		}
		return nil
	}

	if !c.Bool("index") {
		cli.ShowAppHelp(c)
		return nil
	}

	if err := indexer.Run(cfg.DataBase, c.String("prefix"), c.String("bucket"), newSession); err != nil {
		return fmt.Errorf("error in indexing: %v", err)
	}

	return nil
}
