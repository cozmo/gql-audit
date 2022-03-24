package main

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jensneuse/graphql-go-tools/pkg/ast"
	"github.com/jensneuse/graphql-go-tools/pkg/astparser"
	"github.com/jensneuse/graphql-go-tools/pkg/asttransform"
	"github.com/jensneuse/graphql-go-tools/pkg/astvisitor"
	"github.com/jensneuse/graphql-go-tools/pkg/introspection"
	"github.com/jensneuse/graphql-go-tools/pkg/operationreport"
)

func loadSchema(schemaPath string) (*ast.Document, error) {
	// TODO support loading schema definition from a URL

	file, err := os.Open(schemaPath)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(schemaPath, ".json") {
		schema := introspection.JsonConverter{}
		return schema.GraphQLDocument(file)
	} else {
		sdlBytes, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		schema, report := astparser.ParseGraphqlDocumentBytes(sdlBytes)
		if report.HasErrors() {
			return nil, errors.New(report.Error())
		}
		err = asttransform.MergeDefinitionWithBaseSchema(&schema)
		return &schema, err
	}
}

func schemaToTypes(schema *ast.Document) map[string]map[string]bool {
	types := map[string]map[string]bool{}
	for _, graphqlType := range schema.ObjectTypeDefinitions {
		typeName := string(schema.Input.RawBytes[graphqlType.Name.Start:graphqlType.Name.End])
		if _, ok := types[typeName]; !ok {
			types[typeName] = map[string]bool{}
		}
		for _, fieldRef := range graphqlType.FieldsDefinition.Refs {
			types[typeName][schema.FieldDefinitionNameString(fieldRef)] = true
		}
	}

	for _, graphqlType := range schema.InputObjectTypeDefinitions {
		typeName := string(schema.Input.RawBytes[graphqlType.Name.Start:graphqlType.Name.End])
		if _, ok := types[typeName]; !ok {
			types[typeName] = map[string]bool{}
		}
		for _, fieldRef := range graphqlType.InputFieldsDefinition.Refs {
			types[typeName][schema.InputValueDefinitionNameString(fieldRef)] = true
		}
	}

	return types
}

type vistor struct {
	uses              []string
	typeName          string
	fieldName         string
	walker            *astvisitor.Walker
	schemaDocument    *ast.Document
	operationDocument *ast.Document
}

func (v *vistor) EnterField(ref int) {
	if (v.typeName == v.walker.EnclosingTypeDefinition.NameString(v.schemaDocument)) && (v.fieldName == v.operationDocument.FieldNameString(ref)) {
		v.uses = append(v.uses, v.walker.Path.DotDelimitedString()+"."+v.operationDocument.FieldNameString(ref))
	}
}

func findUsesOfTypeInDocument(walker astvisitor.Walker, schema *ast.Document, document *ast.Document, typeName string, fieldName string) (*[]string, error) {
	walker.ResetVisitors()
	v := vistor{
		typeName:          typeName,
		fieldName:         fieldName,
		schemaDocument:    schema,
		operationDocument: document,
		walker:            &walker,
		uses:              []string{},
	}

	walker.RegisterEnterFieldVisitor(&v)

	var report operationreport.Report
	walker.Walk(document, schema, &report)

	if report.HasErrors() {
		return nil, errors.New(report.Error())
	}

	return &v.uses, nil
}
