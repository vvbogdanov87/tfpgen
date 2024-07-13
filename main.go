package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/vvbogdanov87/tfpgen/pkg/config"
	"github.com/vvbogdanov87/tfpgen/pkg/generator"
)

func main() {
	app := &cli.App{
		Name:  "tfpgen",
		Usage: "generate Terraform provider code from Kubernetes CRD schemas",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "tfpgen.yaml",
				Usage: "path to the configuration file",
			},
		},

		Action: func(cCtx *cli.Context) error {
			config, err := config.NewConfig(cCtx.String("config"))
			if err != nil {
				return fmt.Errorf("read tfpgen config: %w", err)
			}

			generator := generator.NewGenerator(config)

			err = generator.Generate()
			if err != nil {
				return fmt.Errorf("generate provider code: %w", err)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("run app", "error", err)
		os.Exit(1)
	}
}
