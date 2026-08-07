package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"ssooj/receipt-worker/receipt"
)

func main() {
	pdfPath := flag.String("pdf", "", "path to PDF file")
	flag.Parse()

	if *pdfPath == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/local.go --pdf <path>")
		os.Exit(1)
	}

	text, err := extractText(*pdfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== RAW TEXT ===")
	fmt.Println(text)
	fmt.Println("=== PARSED ===")

	r, err := receipt.Parse(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	out, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(out))
}

func extractText(path string) (string, error) {
	tmp := path + ".txt"
	cmd := exec.Command("pdftotext", "-layout", path, tmp)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	data, err := os.ReadFile(tmp)
	os.Remove(tmp)
	return string(data), err
}
