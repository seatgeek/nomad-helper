package tail

import (
	"fmt"
	"os"
)

type rawLogWriter struct {
	Type string
}

func (w rawLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)

	if w.Type == "stdout" {
		fmt.Fprintf(os.Stdout, s)
	} else {
		fmt.Fprintf(os.Stderr, s)
	}

	return len(p), nil
}
