package tail

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

type colorLogWriter struct {
	Type string
}

func (w colorLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)

	if string(s[0]) == "{" {
		buffer := bytes.NewBufferString("")
		highlight(buffer, s, "json", "terminal", "emacs")
		s = buffer.String()
	}

	if w.Type == "stdout" {
		fmt.Fprintf(os.Stdout, s)
	} else {
		fmt.Fprintf(os.Stderr, s)
	}

	return len(p), nil
}

func highlight(w io.Writer, source, lexer, formatter, style string) error {
	// Determine lexer.
	l := lexers.Get(lexer)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	// Determine formatter.
	f := formatters.Get(formatter)
	if f == nil {
		f = formatters.Fallback
	}

	// Determine style.
	s := styles.Get(style)
	if s == nil {
		s = styles.Fallback
	}

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return f.Format(w, s, it)
}
