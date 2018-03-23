package driver

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sapk/docker-volume-helpers/basic"
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

//GetMountName get moint point base on request and driver config (mountUniqName)
func GetMountName(d *basic.Driver, r *volume.CreateRequest) string {
	if d.Config.CustomOptions["mountUniqName"].(bool) {
		return url.PathEscape(r.Options["voluri"])
	}
	return url.PathEscape(r.Name)
}
