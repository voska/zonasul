package outfmt_test

import (
	"bytes"
	"testing"

	"github.com/mattvoska/zonasul/internal/outfmt"
)

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	f := outfmt.New(true, false, &buf)
	data := map[string]string{"name": "Banana"}
	if err := f.Print(data); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if got != "{\"name\":\"Banana\"}\n" {
		t.Errorf("got %q", got)
	}
}

func TestHuman(t *testing.T) {
	var buf bytes.Buffer
	f := outfmt.New(false, false, &buf)
	if err := f.Print("hello"); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "hello\n" {
		t.Errorf("got %q", buf.String())
	}
}
