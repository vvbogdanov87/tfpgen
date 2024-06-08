package main

import (
	"fmt"
	"go/format"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
)

func main() {
	logger := slog.Default()

	// TODO: handle working directory when started from a different locations (e.g. vscode debug vs make generate)
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("get working directory: ", err)
		os.Exit(1)
	}

	tmplName := "crd.tmpl"
	tmplFilePath := filepath.Join(cwd, "templates", tmplName)

	tmpl, err := template.New(tmplName).ParseFiles(tmplFilePath)
	if err != nil {
		logger.Error("parse template: ", err)
		os.Exit(1)
	}

	schemasDir := filepath.Join(cwd, "../../schemas")
	err = filepath.WalkDir(schemasDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk schemas directory: %w", err)
		}

		if d.IsDir() {
			return nil
		}

		logger.Info("parsing schema", "path", path)
		data := parseSchema(path)

		// generate code
		outFilePath := filepath.Join(cwd, "../../out", data.FileName)
		outFile, err := os.Create(outFilePath)
		if err != nil {
			logger.Error("create output file: ", err)
		}

		err = tmpl.Execute(outFile, data)
		if err != nil {
			logger.Error("execute template: ", err)
			os.Exit(1)
		}

		//format code
		unformatted, err := os.ReadFile(outFilePath)
		if err != nil {
			return fmt.Errorf("read unformatted file %s: %w", outFilePath, err)
		}
		formatted, err := format.Source(unformatted)
		if err != nil {
			return fmt.Errorf("format source %s: %w", outFilePath, err)
		}
		err = os.WriteFile(outFilePath, formatted, 0644)
		if err != nil {
			return fmt.Errorf("write formatted file %s: %w", outFilePath, err)
		}

		return nil
	})
	if err != nil {
		logger.Error("generating CRD types", err)
		os.Exit(1)
	}

}
