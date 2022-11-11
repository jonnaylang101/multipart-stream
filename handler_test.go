package multipartstream

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/labstack/echo"
)

var (
	loremContent, _ = os.ReadFile("./files/in.txt")
	vals            = map[string][]string{
		"name": {"Jobby"},
		"fave": {"Wotsits"},
	}
)

func TestHandler(t *testing.T) {
	file, _ := os.Open("./files/in.txt")
	out, _ := os.Create("./files/out.txt")
	type setup struct {
		bdy io.ReadSeeker
	}
	type args struct {
		req  *http.Request
		vals map[string][]string
		wrt  io.ReadWriter
		pl   ProgressListener
	}
	tests := []struct {
		name      string
		setup     setup
		args      args
		want      []byte
		wantErr   bool
		wantVals  map[string][]string
		wantPanic bool
	}{
		{
			name: "When the http request has a nil value it should let us know via an error",
			setup: setup{
				bdy: file,
			},
			args: args{
				req:  nil,
				vals: make(map[string][]string),
				wrt:  out,
				pl:   nil,
			},
			want:      []byte{},
			wantErr:   true,
			wantVals:  map[string][]string{},
			wantPanic: true,
		},
		{
			name: "When the request has no multipart body",
			setup: setup{
				bdy: nil,
			},
			args: args{
				req:  &http.Request{},
				vals: make(map[string][]string),
				wrt:  out,
				pl:   nil,
			},
			want:     []byte{},
			wantErr:  true,
			wantVals: map[string][]string{},
		},
		{
			name: "When the request is good",
			setup: setup{
				bdy: file,
			},
			args: args{
				req:  &http.Request{},
				vals: make(map[string][]string),
				wrt:  bytes.NewBuffer([]byte{}),
				pl:   nil,
			},
			want:     loremContent,
			wantErr:  false,
			wantVals: vals,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if panErr := recover(); (panErr != nil) != tt.wantPanic {
					t.Errorf("expected panic to be %v but got %v", tt.wantPanic, panErr)
					return
				}
			}()
			file.Seek(0, 0)
			mpBody, ct, err := createMultipart(tt.setup.bdy, "body.txt")
			if err != nil {
				t.Fatal(err)
			}

			testReq := httptest.NewRequest("Post", "/", mpBody)
			testReq.Header.Set(echo.HeaderContentType, ct)

			if tt.args.req == nil {
				testReq = nil
			}

			binder, newErr := NewBinder(testReq, DefaultValuesBytesize)
			if (newErr != nil) != tt.wantErr {
				t.Errorf("wanted err to be %v but got %v\n", tt.wantErr, err)
				return
			}

			if newErr != nil {
				return
			}

			err = binder.Bind(tt.args.wrt, tt.args.pl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			buf := make([]byte, 1024*1024*2)
			n, err := tt.args.wrt.Read(buf)
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}

			_got := buf[:n]
			if !reflect.DeepEqual(_got, tt.want) {
				t.Errorf("Handler() = %v, want %v", _got, tt.want)
				return
			}

			if !reflect.DeepEqual(binder.Values(), tt.wantVals) {
				t.Errorf("Expected vals to be \n%+v\n but got \n%+v\n", tt.wantVals, binder.Values())
			}
		})
	}
}

func createMultipart(file io.ReadSeeker, filename string) (rdr io.Reader, ct string, err error) {
	buf := bytes.NewBuffer(nil)
	w := multipart.NewWriter(buf)

	if err = w.WriteField("name", "Jobby"); err != nil {
		return
	}
	if err = w.WriteField("fave", "Wotsits"); err != nil {
		return
	}

	if file == nil {
		return
	}

	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return
	}
	n, _ := io.Copy(part, file)
	fmt.Println(n)

	ct = w.FormDataContentType()

	if err = w.Close(); err != nil {
		return
	}

	rdr = buf

	return
}
