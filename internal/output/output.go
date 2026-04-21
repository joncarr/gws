package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
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

func (w Writer) Table(rows [][]string) error {
	tw := tabwriter.NewWriter(w.out, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func (w Writer) CSV(rows [][]string) error {
	cw := csv.NewWriter(w.out)
	if err := cw.WriteAll(rows); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}
