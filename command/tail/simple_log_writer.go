package tail

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

type simpleLogWriter struct {
	Type string
}

func (w simpleLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	s = strings.Trim(s, "\n")

	if w.Type == "stdout" {
		log.Info(s)
	} else if w.Type == "stderr" {
		log.Error(s)
	} else {
		log.Warn(s)
	}

	return len(p), nil
}
