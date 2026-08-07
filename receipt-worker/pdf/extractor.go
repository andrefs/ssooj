package pdf

import (
	"bytes"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func ExtractText(data []byte) (string, error) {
	var result string

	digest := func(r io.Reader, _ int) error {
		b, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		result += string(b) + "\n"
		return nil
	}

	conf := model.NewDefaultConfiguration()
	conf.Cmd = model.EXTRACTCONTENT

	err := api.ExtractContent(bytes.NewReader(data), nil, digest, conf)
	if err != nil {
		return "", err
	}

	return result, nil
}
