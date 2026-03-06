package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Formatter struct {
	json   bool
	plain  bool
	writer io.Writer
}

func New(jsonMode, plainMode bool, w io.Writer) *Formatter {
	return &Formatter{json: jsonMode, plain: plainMode, writer: w}
}

func FromGlobals(jsonMode, plainMode bool) *Formatter {
	return New(jsonMode, plainMode, os.Stdout)
}

func (f *Formatter) Print(v any) error {
	if f.json {
		enc := json.NewEncoder(f.writer)
		return enc.Encode(v)
	}
	_, err := fmt.Fprintln(f.writer, v)
	return err
}

func (f *Formatter) IsJSON() bool { return f.json }
