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
	err = filepath.WalkDir(schemasDir, func(schemaPath string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk schemas directory: %w", err)
		}

		if d.IsDir() {
			return nil
		}

		logger.Info("generating code for schema", "path", schemaPath)

		data, err := parseSchema(schemaPath)
		if err != nil {
			return fmt.Errorf("parse schema: %w", err)
		}

		outFilePath := filepath.Join(cwd, "../../out", data.FileName)

		err = generateCode(outFilePath, tmpl, data)
		if err != nil {
			return fmt.Errorf("generate code: %w", err)
		}

		err = formatCode(outFilePath)
		if err != nil {
			return fmt.Errorf("format code: %w", err)
		}

		return nil
	})
	if err != nil {
		logger.Error("generating CRD types", err)
		os.Exit(1)
	}
}

func generateCode(filePath string, tmpl *template.Template, data Data) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

func formatCode(filePath string) error {
	unformatted, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read unformatted file %s: %w", filePath, err)
	}

	formatted, err := format.Source(unformatted)
	if err != nil {
		return fmt.Errorf("format source %s: %w", filePath, err)
	}

	err = os.WriteFile(filePath, formatted, 0644)
	if err != nil {
		return fmt.Errorf("write formatted file %s: %w", filePath, err)
	}

	return nil
}
