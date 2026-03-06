package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/muesli/termenv"
)

var profile = termenv.ColorProfile()

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

func Hint(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	styled := termenv.String(msg).Foreground(profile.Color("8"))
	fmt.Fprintln(os.Stderr, styled)
}

func Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	styled := termenv.String(msg).Foreground(profile.Color("2"))
	fmt.Fprintln(os.Stderr, styled)
}

func Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	styled := termenv.String(msg).Foreground(profile.Color("3"))
	fmt.Fprintln(os.Stderr, styled)
}

func ErrorMsg(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	styled := termenv.String(msg).Foreground(profile.Color("1"))
	fmt.Fprintln(os.Stderr, styled)
}
