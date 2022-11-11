package multipartstream

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

const (
	DefaultValuesBytesize = int64(10 << 20) // 10 MB
)

type Binder interface {
	Bind(io.Writer, ProgressListener) error
	Values() map[string][]string
}

type binder struct {
	mr                *multipart.Reader
	vals              url.Values
	maxValuesBytesize int64
}

func NewBinder(r *http.Request, maxValuesBytesize int64) (Binder, error) {
	if r == nil {
		panic("NewBinder: provided r (*http.Request) param has nil value")
	}

	mr, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("NewBinder: unable to open multipart reader >> %w", err)
	}

	if maxValuesBytesize == 0 {
		maxValuesBytesize = DefaultValuesBytesize
	}

	return &binder{
		mr:                mr,
		vals:              make(url.Values),
		maxValuesBytesize: maxValuesBytesize,
	}, nil
}

func (b *binder) Values() map[string][]string {
	return b.vals
}

func (b *binder) Bind(dst io.Writer, pl ProgressListener) error {
	for {
		part, err := b.mr.NextPart()
		if err == io.EOF {
			break
		}

		name := part.FormName()
		if name == "" {
			continue
		}

		if partIsForm(part) {
			if err := b.bindFormPart(part, name); err != nil {
				return fmt.Errorf("Bind >> %w", err)
			}
			continue
		}

		copyFilePart(part, dst)
	}

	return nil
}

func (b *binder) bindFormPart(part *multipart.Part, key string) error {
	buf := bytes.NewBuffer([]byte{})

	bytesCopied, err := io.CopyN(buf, part, b.maxValuesBytesize)
	if err != nil && err != io.EOF {
		return errors.New("bindFormPart: error processing part")
	}

	b.maxValuesBytesize -= bytesCopied
	if b.maxValuesBytesize == 0 {
		return errors.New("bindFormPart: multipart part too large")
	}

	b.vals.Add(key, buf.String())

	return nil
}

func copyFilePart(part *multipart.Part, dst io.Writer) {
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
