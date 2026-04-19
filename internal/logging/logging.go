package logging

import (
	"fmt"
	"io"
)

func Error(w io.Writer, err error) {
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
	}
}
