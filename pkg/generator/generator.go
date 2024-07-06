package generator

import (
	"embed"
	"fmt"
	"go/format"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/vvbogdanov87/tfpgen/pkg/config"
)

//go:embed templates/crd.go.tmpl
//go:embed templates/crd_property.go.tmpl
var crdTemplates embed.FS

//go:embed templates/resource.go.tmpl
//go:embed templates/schema_attribute.go.tmpl
var resourceTmplates embed.FS

//go:embed templates/resources.go.tmpl
var resourcesTemplate embed.FS

type Generator struct {
	config *config.Config
}

func NewGenerator(config *config.Config) *Generator {
	return &Generator{
		config: config,
	}
}

func (g *Generator) Generate(config *config.Config) error {
	packages, err := g.generateResources()
	if err != nil {
		return fmt.Errorf("generate resources: %w", err)
	}

	err = g.generateProviderResources(packages)
	if err != nil {
		return fmt.Errorf("generate provider resources method: %w", err)
	}

	return nil
}

func (g *Generator) generateResources() ([]string, error) {
	crdTmpl, err := template.ParseFS(crdTemplates, "templates/crd.go.tmpl", "templates/crd_property.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("get crd template: %w", err)
	}

	resourceTmpl, err := template.ParseFS(resourceTmplates, "templates/resource.go.tmpl", "templates/schema_attribute.go.tmpl")
	if err != nil {
		return nil, fmt.Errorf("get resource template: %w", err)
	}

	var packages []string

	// generate code for each schema from each template
	err = filepath.WalkDir(g.config.SchemasDir, func(schemaPath string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk schemas directory: %w", err)
		}

		if d.IsDir() {
			return nil
		}

		slog.Info("generating code for schema", "path", schemaPath)

		data, err := parseSchema(schemaPath)
		if err != nil {
			return fmt.Errorf("parse schema: %w", err)
		}

		outDir := filepath.Join(g.config.OutputDir, "/internal/provider", data.PackageName)
		err = os.MkdirAll(outDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}

		err = generateCode(crdTmpl, data, outDir, "crd.go")
		if err != nil {
			return fmt.Errorf("generate CRD code: %w", err)
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

func (g *Generator) generateProviderResources(packages []string) error {
	tmpl, err := template.ParseFS(resourcesTemplate, "templates/resources.go.tmpl")
	if err != nil {
		return fmt.Errorf("get provider resources template: %w", err)
	}

	outDir := filepath.Join(g.config.OutputDir, "internal/provider")

	err = generateCode(tmpl, packages, outDir, "resources.go")
	if err != nil {
		return fmt.Errorf("generate provider resources method code: %w", err)
	}

	return nil
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
