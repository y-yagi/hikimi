package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	_ "github.com/mattn/go-sqlite3"
	"github.com/urfave/cli/v2"
	"github.com/y-yagi/configure"
	"github.com/y-yagi/hikimi/downloader"
	"github.com/y-yagi/hikimi/identifier"
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
			Name:    "region",
			Aliases: []string{"r"},
			Usage:   "region of Wasabi",
			Value:   "us-east-1",
		},
		&cli.StringFlag{
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
		&cli.StringFlag{
			Name:  "download",
			Usage: "dowload files",
		},
		&cli.StringSliceFlag{
			Name:  "identify",
			Usage: "identify a local file and an uploaded file",
		},
	}
}

func appRun(c *cli.Context) error {
	if c.Bool("config") {
		return configure.Edit("hikimi", "vim")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.String("accesskey"),
			c.String("secret"),
			"",
		)),
		awsconfig.WithRegion(c.String("region")),
		awsconfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationUnset),
	)
	if err != nil {
		return err
	}

	// Configure for Wasabi
	awsCfg.BaseEndpoint = aws.String("https://s3.wasabisys.com")

	if len(c.String("search")) != 0 {
		if err := searcher.Run(c, cfg.DataBase, cfg.DownloadPath, awsCfg); err != nil {
			fmt.Printf("error in search: %v", err)
			return err
		}
		return nil
	}

	if len(c.String("download")) != 0 {
		if err := downloader.Run(c.String("bucket"), c.String("download"), cfg.DownloadPath, awsCfg); err != nil {
			fmt.Printf("error in downloading: %v", err)
			return err
		}

		return nil
	}

	if len(c.StringSlice("identify")) != 0 {
		files := c.StringSlice("identify")
		if len(files) != 2 {
			return errors.New("pleaes specify a local file path and an uploaded file path")
		}

		if err := identifier.Run(c.String("bucket"), files[0], files[1], awsCfg); err != nil {
			fmt.Printf("error in identifying: %v", err)
			return err
		}
		return nil
	}

	if len(c.String("index")) == 0 {
		cli.ShowAppHelp(c)
		return nil
	}

	if err := indexer.Run(cfg.DataBase, c.String("index"), c.String("bucket"), awsCfg); err != nil {
		return fmt.Errorf("error in indexing: %v", err)
	}

	return nil
}
