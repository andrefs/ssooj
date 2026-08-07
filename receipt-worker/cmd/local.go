// Build: go run ./cmd/local.go --pdf ~/Downloads/receipt.pdf

func main() {
    pdfPath := flag.String("pdf", "", "path to PDF file")
    flag.Parse()
    data, _ := os.ReadFile(*pdfPath)
    text, _ := pdf.ExtractText(data)       // same function Lambda uses
    receipt, _ := receipt.Parse(text)       // same function Lambda uses
    json, _ := json.MarshalIndent(receipt, "", "  ")
    fmt.Println(string(json))
}
