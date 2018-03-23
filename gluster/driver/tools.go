package driver

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	validHostnameRegex = `(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])`
)

func isValidURI(volURI string) bool {
	re := regexp.MustCompile(validHostnameRegex + ":.+")
	return re.MatchString(volURI)
}

func parseVolURI(volURI string) string {
	volParts := strings.Split(volURI, ":")
	volServers := strings.Split(volParts[0], ",")
	return fmt.Sprintf("--volfile-id='%s' -s '%s'", volParts[1], strings.Join(volServers, "' -s '"))
}
