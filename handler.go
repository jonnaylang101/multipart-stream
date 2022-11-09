package multipartstream

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

func Handler(r *http.Request, vals map[string][]string, dst io.Writer, pl ProgressListener) error {
	if r == nil {
		return fmt.Errorf("multipartstream.Handler: provided req param has nil value")
	}

	if vals == nil {
		return fmt.Errorf("multipartstream.Handler: provided vals param has nil value")
	}

	mr, err := r.MultipartReader()
	if err != nil {
		return fmt.Errorf("multipartstream.Handler: unable to open multipart reader >> %w", err)
	}

	maxValueBytes := int64(10 << 20)
	for {

		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}

		name := part.FormName()
		if name == "" {
			continue
		}

		if partIsForm(part) {
			val, err := parseFormPart(part, maxValueBytes)
			if err != nil {
				return fmt.Errorf("multipartstream.Handler: >> %w", err)
			}

			vals[name] = append(vals[name], val)
			continue
		}

		parseFilePart(part, dst)
	}

	return nil
}

func parseFormPart(part *multipart.Part, max int64) (string, error) {
	buf := bytes.NewBuffer([]byte{})

	bytesCopied, err := io.CopyN(buf, part, max)
	if err != nil && err != io.EOF {
		return "", errors.New("parseFormPart: error processing part")
	}

	max -= bytesCopied
	if max == 0 {
		return "", errors.New("parseFormPart: multipart part too large")
	}

	return buf.String(), nil
}

func parseFilePart(part *multipart.Part, dst io.Writer) {
	for {
		buffer := make([]byte, 100000)
		cBytes, err := part.Read(buffer)
		dst.Write(buffer[0:cBytes])
		if err == io.EOF {
			break
		}
	}
}

func partIsForm(part *multipart.Part) bool {
	return part.FileName() == ""
}
