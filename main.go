package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/soh335/mtexport/ast"
	"github.com/soh335/mtexport/parser"
)

var (
	input     = flag.String("input", "", "input")
	outputDir = flag.String("outputDir", "content", "outputDir")
	timezone  = flag.String("timezone", "Asia/Tokyo", "timezone")
)

var contentTmpl = template.Must(parseTemplate())
var location *time.Location

type Data struct {
	Basename string
	Tags     []string
	Draft    bool
	Title    string
	Date     time.Time
	Content  string
}

func (d *Data) TagsAsString() string {
	var quotedTags []string
	for _, tag := range d.Tags {
		quotedTags = append(quotedTags, fmt.Sprintf("\"%v\"", tag))
	}
	return fmt.Sprintf("[%s]", strings.Join(quotedTags, ", "))
}

func (d *Data) MarkdownFilename(outputDir string) string {
	return filepath.Join(outputDir, fmt.Sprintf("%s.md", d.Basename))
}

func main() {
	flag.Parse()
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

func _main() error {

	var err error
	location, err = time.LoadLocation(*timezone)
	if err != nil {
		return err
	}

	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()

	stmts, err := parser.Parse(f, []string{"IMAGE"})
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		if err := genOutput(stmt, *outputDir); err != nil {
			log.Println("got error:", err)
		}
	}

	return nil
}

func parseTemplate() (*template.Template, error) {
	return template.New("contentTmpl").Parse(`+++
date  = "{{ .Date.Format "2006-01-02T15:04:05-07:00" }}"
draft = {{ .Draft }}
title = "{{ .Title }}"
tags  = {{ .TagsAsString }}
+++
{{ .Content }}
`)
}

func genOutput(stmt ast.Stmt, outputDir string) error {
	data, err := parseStmt(stmt)
	if err != nil {
		return err
	}
	output := data.MarkdownFilename(outputDir)
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	o, err := os.Create(output)
	if err != nil {
		return err
	}
	defer o.Close()
	if err := contentTmpl.Execute(o, data); err != nil {
		return err
	}
	return nil
}

func parseStmt(stmt ast.Stmt) (*Data, error) {
	entry := stmt.(*ast.EntryStmt)

	data := &Data{}

	for _, _stmt := range entry.SectionStmts {
		switch _stmt.(type) {
		case *ast.NormalSectionStmt:
			err := parseFieldSection(_stmt.(*ast.NormalSectionStmt), data)
			if err != nil {
				return nil, err
			}
		case *ast.MultilineSectionStmt:
			_stmt := _stmt.(*ast.MultilineSectionStmt)
			switch _stmt.Key {
			case "BODY":
				data.Content = _stmt.Body
			case "COMMENT":
				log.Println("COMMENT is ignored")
			case "EXCERPT":
				log.Println("EXCERPT is ignored")
			default:
				return nil, fmt.Errorf("not supported multiline section key: %v", _stmt.Key)
			}
		default:
			panic("not reach")
		}
	}

	return data, nil
}

func parseFieldSection(stmt *ast.NormalSectionStmt, data *Data) error {
	for _, _stmt := range stmt.FieldStmts {
		field, ok := _stmt.(*ast.FieldStmt)
		if !ok {
			return nil
		}
		value := strings.TrimSpace(field.Value)
		switch field.Key {
		case "TITLE":
			data.Title = value
		case "STATUS":
			switch strings.ToLower(value) {
			case "draft":
				data.Draft = true
			case "publish":
				data.Draft = false
			default:
				return fmt.Errorf("not supported status: %v", value)
			}
		case "DATE":
			t, err := time.ParseInLocation("01/02/2006 15:04:05", value, location)
			if err != nil {
				return err
			}
			data.Date = t
		case "BASENAME":
			data.Basename = value
		case "CATEGORY":
			data.Tags = append(data.Tags, value)
		default:
		}
	}

	return nil
}
