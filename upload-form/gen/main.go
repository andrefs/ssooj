package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type envFile struct {
	API_ID string
}

func loadEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	env := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return env, sc.Err()
}

func main() {
	tmplPath := flag.String("template", "index.tmpl.html", "path to the HTML template")
	envPath := flag.String("env", ".env", "path to the .env file")
	outPath := flag.String("out", "dist/index.html", "path to the generated HTML file")
	flag.Parse()

	env, err := loadEnv(*envPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %v\n", *envPath, err)
		os.Exit(1)
	}

	apiID := strings.TrimSpace(env["API_ID"])
	if apiID == "" {
		fmt.Fprintln(os.Stderr, "error: API_ID is not set in "+*envPath)
		os.Exit(1)
	}
	if apiID == "xxxxxxxxxx" {
		fmt.Fprintln(os.Stderr, "error: API_ID in "+*envPath+" is still the placeholder value")
		os.Exit(1)
	}

	tmpl, err := template.ParseFiles(*tmplPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing template: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, envFile{API_ID: apiID}); err != nil {
		fmt.Fprintf(os.Stderr, "error rendering template: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("generated %s (API_ID=%s)\n", *outPath, apiID)
}
