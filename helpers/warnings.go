package helpers

import (
	"fmt"
	"strings"
)

func WarningFormat(es []error) string {
	if len(es) == 1 {
		return fmt.Sprintf("1 warning occurred:\n\n* %s", es[0])
	}

	points := make([]string, len(es))
	for i, err := range es {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d warnings occurred:\n\n%s",
		len(es), strings.Join(points, "\n"))
}
