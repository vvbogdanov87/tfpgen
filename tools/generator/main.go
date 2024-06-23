package main

import (
	"fmt"
	"go/format"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"
)

var logger = slog.Default()

func main() {
	// logger := slog.Default()

	// TODO: handle working directory when started from a different locations (e.g. vscode debug vs make generate)
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("get working directory: ", err)
		os.Exit(1)
	}

	packages, err := generateResources(cwd)
	if err != nil {
		logger.Error("generate resources: ", err)
		os.Exit(1)
	}

	err = generateProviderResources(cwd, packages)
	if err != nil {
		logger.Error("generate provider resources method: ", err)
		os.Exit(1)
	}
}

func generateResources(cwd string) ([]string, error) {
	resourcesPath := "internal/provider/resources"

	crdTmpl, err := getTemplate(cwd, resourcesPath, "crd.go.tmpl", "crd_property.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("get crd template: %w", err)
	}
	modelTmpl, err := getTemplate(cwd, resourcesPath, "model.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("get model template: %w", err)
	}
	resourceTmpl, err := getTemplate(cwd, resourcesPath, "resource.go.tmpl", "schema_attribute.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("get resource template: %w", err)
	}

	var packages []string

	// generate code for each schema from each template
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

		outDir := filepath.Join(cwd, "../../internal/provider", data.PackageName)
		err = os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}

		err = generateCode(crdTmpl, data, outDir, "crd.go")
		if err != nil {
			return fmt.Errorf("generate CRD code: %w", err)
		}

		err = generateCode(modelTmpl, data, outDir, "model.go")
		if err != nil {
			return fmt.Errorf("generate Terraform Resource Model code: %w", err)
		}

		err = generateCode(resourceTmpl, data, outDir, "resource.go")
		if err != nil {
			return fmt.Errorf("generate Terraform resource code: %w", err)
		}

		packages = append(packages, data.PackageName)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("generating CRD types: %w", err)
	}

	return packages, nil
}

func generateProviderResources(cwd string, packages []string) error {
	tmpl, err := getTemplate(cwd, "internal/provider", "resources.go.tmpl")
	if err != nil {
		return fmt.Errorf("get crd template: %w", err)
	}

	outDir := filepath.Join(cwd, "../../internal/provider")

	err = generateCode(tmpl, packages, outDir, "resources.go")
	if err != nil {
		return fmt.Errorf("generate provider resources method code: %w", err)
	}

	return nil
}

func getTemplate(cwd, tmplPath string, tmplNames ...string) (*template.Template, error) {
	var tmplFilePaths []string
	for _, name := range tmplNames {
		tmplFilePaths = append(tmplFilePaths, filepath.Join(cwd, "templates", tmplPath, name))
	}
	return template.New(tmplNames[0]).ParseFiles(tmplFilePaths...)
}

func generateCode(tmpl *template.Template, data any, outDir, outFileName string) error {
	outFilePath := filepath.Join(outDir, outFileName)

	err := executeTemplate(outFilePath, tmpl, data)
	if err != nil {
		return fmt.Errorf("generate code: %w", err)
	}

	err = formatCode(outFilePath)
	if err != nil {
		return fmt.Errorf("format code: %w", err)
	}

	return nil
}

func executeTemplate(filePath string, tmpl *template.Template, data any) error {
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
