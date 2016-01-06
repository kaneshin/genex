// Command mode parses a JSON to generate.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/kaneshin/genetron"
	"github.com/serenize/snaker"
)

var (
	pkg    = flag.String("pkg", "", "Package name to use in the generated code. (default \"main\")")
	srcDir = flag.String("path", "", "output file directory")
	output = flag.String("output", "", "output file name; default srcdir/mode_gen.go")
)

const (
	NAME   = "mode"
	OUTPUT = "mode_gen.go"
)

// Usage is a replacement usage function for the flags package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(fmt.Sprintf("gen-%s: ", NAME))
	flag.Usage = Usage
	flag.Parse()
	if len(*pkg) == 0 {
		*pkg = "main"
	}

	// We accept either one directory or a list of files. Which do we have?
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(3)
	}

	if len(*srcDir) > 0 {
		if d, err := filepath.Abs(*srcDir); err == nil {
			*srcDir = d
		}
	}

	if len(*srcDir) == 0 {
		*srcDir = filepath.Dir(os.Args[0])
	}

	os.Exit(run())
}

func run() int {
	var (
		args = flag.Args()
		g    genetron.Generator
	)

	// Print the header and package clause.
	g.Printf("// Code generated by gen-mode.\n// %s\n// DO NOT EDIT\n", strings.Join(args, "\n// "))
	g.Printf("\n")
	g.Printf("package %s", *pkg)
	g.Printf("\n")

	// Run generate for each type.
	jsonData := map[string]interface{}{}
	for _, name := range args {
		b, err := ioutil.ReadFile(name)
		if err != nil {
			panic(err)
		}
		json.Unmarshal(b, &jsonData)
	}

	// Generate two source files.
	params := map[string]interface{}{}

	type Element struct {
		CamelLetters, CamelLettersMode, Letters, Meta string
	}
	data := []Element{}
	for _, vv := range jsonData["data"].([]interface{}) {
		v := vv.(map[string]interface{})
		elm := Element{}
		if r, found := v["value"]; found {
			if str, ok := r.(string); ok {
				elm.CamelLetters = snaker.SnakeToCamel(str)
				elm.CamelLettersMode = elm.CamelLetters + "Mode"
				elm.Letters = str
			}
		}
		if r, found := v["meta"]; found {
			if str, ok := r.(string); ok {
				elm.Meta = str
			}
		}
		if r, found := v["default"]; found {
			if def, ok := r.(bool); ok && def {
				params["default"] = elm.CamelLettersMode
			}
		}
		data = append(data, elm)
	}

	// default
	if _, found := params["default"]; !found {
		params["default"] = data[0].CamelLettersMode
	}
	params["data"] = data

	// Format the output.
	t := template.Must(template.New("").Parse(text))
	var buf bytes.Buffer
	t.Execute(&buf, params)
	g.Printf("%s", buf.String())
	src := g.Format()

	// Write to file.
	outputName := *output
	if outputName == "" {
		baseName := OUTPUT
		outputName = filepath.Join(*srcDir, strings.ToLower(baseName))
	}
	err := ioutil.WriteFile(outputName, src, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}

	return 0
}

const text = `
import (
	"fmt"
	"os"
)

const ENV_MODE = "MODE"

const ({{range .data}}
	{{.CamelLettersMode}} string = "{{.Letters}}"{{end}}
)

var modeName string = {{.default}}

func Mode() string {
	return modeName
}

func init() {
	mode := os.Getenv(ENV_MODE)
	if len(mode) == 0 {
		SetMode({{.default}})
	} else {
		SetMode(mode)
	}
}

func SetMode(value string) {
	switch value { {{range .data}}
	case {{.CamelLettersMode}}:
		{{.Meta}}{{end}}
	default:
		panic("mode unknown: " + value)
	}
	modeName = value
}
{{range .data}}
func Is{{.CamelLetters}}() bool {
	return Mode() == {{.CamelLettersMode}}
}
{{end}}`