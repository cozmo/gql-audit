package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jensneuse/graphql-go-tools/pkg/ast"
	"github.com/jensneuse/graphql-go-tools/pkg/astparser"
	"github.com/jensneuse/graphql-go-tools/pkg/astvisitor"
	"github.com/jensneuse/graphql-go-tools/pkg/operationreport"
	"github.com/jessevdk/go-flags"
	"github.com/yargevad/filepathx"
)

type args struct {
	SchemaPath string `short:"s" long:"schema-path" description:"Path to the graphql schema file (can be JSON from an introspection query OR an SDL file)." required:"true"`
	FieldPath  string `short:"f" long:"field-path" description:"Type and field path to search for. Should be in the form 'TypeName.fieldName'." required:"true"`
	Positional struct {
		Paths []string `positional-arg-name:"PATHs"`
	} `positional-args:"true" required:"true"`
}

func exitWithErrorString(err string) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	var args args
	_, err := flags.Parse(&args)

	if e, ok := err.(*flags.Error); ok {
		if e.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	fieldPathParts := strings.Split(args.FieldPath, ".")
	if len(fieldPathParts) != 2 {
		exitWithErrorString("Field path must be in the form 'TypeName.fieldName'")
	}
	typeName := fieldPathParts[0]
	fieldName := fieldPathParts[1]

	schema, err := loadSchema(args.SchemaPath)
	if err != nil {
		exitWithErrorString(fmt.Sprintf("Error loading schema: %s", err))
	}

	validTypes := schemaToTypes(schema)

	if _, ok := validTypes[typeName]; !ok {
		exitWithErrorString(fmt.Sprintf("type `%s` not found in schema", typeName))
	}
	if _, ok := validTypes[typeName][fieldName]; !ok {
		exitWithErrorString(fmt.Sprintf("field `%s` not found in type `%s`", fieldName, typeName))
	}

	paths := []string{}
	for _, globPath := range args.Positional.Paths {
		matches, err := filepathx.Glob(globPath)
		if err != nil {
			exitWithErrorString(fmt.Sprintf("Error parsing PATHs glob: %s", err))
		}
		paths = append(paths, matches...)
	}

	var parseReport operationreport.Report
	parser := astparser.NewParser()
	walker := astvisitor.NewWalker(8)

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			exitWithErrorString(fmt.Sprintf("Error opening file: %s", path))
		}
		defer file.Close()

		bodyBytes, err := ioutil.ReadAll(file)
		if err != nil {
			exitWithErrorString(fmt.Sprintf("Error reading file: %s", path))
		}

		operationDocument := ast.NewDocument()
		operationDocument.Input.ResetInputBytes(bodyBytes)
		parseReport.Reset()
		parser.Parse(operationDocument, &parseReport)

		if parseReport.HasErrors() {
			exitWithErrorString(fmt.Sprintf("Error parsing file: %s: %s", path, parseReport.Error()))
		}

		uses, err := findUsesOfTypeInDocument(walker, schema, operationDocument, typeName, fieldName)
		if err != nil {
			exitWithErrorString(fmt.Sprintf("Error searching for uses: %s", err))
		}

		if uses == nil || len(*uses) == 0 {
			continue
		}

		fmt.Printf("%s:\n", path)
		for _, use := range *uses {
			// TODO: add line numbers, or at least more context about the usage
			fmt.Printf("  %s\n", use)
		}
	}
}
