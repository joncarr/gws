package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type Writer struct {
	out io.Writer
}

func New(out io.Writer) Writer {
	return Writer{out: out}
}

func (w Writer) Printf(format string, args ...any) {
	fmt.Fprintf(w.out, format, args...)
}

func (w Writer) Println(args ...any) {
	fmt.Fprintln(w.out, args...)
}

func (w Writer) JSON(v any) error {
	enc := json.NewEncoder(w.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
