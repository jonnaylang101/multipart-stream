package multipartstream

type State int

const (
	Streaming State = iota
	Completed
)

type ProgressListener interface{}

type Progress struct {
	State         State
	BytesStreamed int64
}
