package pdf

import (
	"bytes"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func ExtractText(data []byte) (string, error) {
	conf := pdfcpu.NewDefaultConfiguration()
	conf.Cmd = pdfcpu.EXTRACT

	msg := ""
	handler := func(page int, text string) {
		msg += text + "\n"
	}

	err := api.ExtractPagesRaw(bytes.NewReader(data), conf, handler)
	if err != nil {
		return "", err
	}
	return msg, nil
}
