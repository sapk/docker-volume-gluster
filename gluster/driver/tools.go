package driver

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sapk/docker-volume-helpers/basic"
	"golang.org/x/net/idna"
)

const (
	validHostnameRegex = `(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9\-])\.)*([A-Za-z0-9\-]|[A-Za-z0-9\-][A-Za-z0-9\-]*[A-Za-z0-9\-])`
	validVolURIRegex   = `((` + validHostnameRegex + `)(,` + validHostnameRegex + `)*):\/?([^\/]+)(/.+)?`
)

func isValidURI(volURI string) bool {
	volURI, err := idna.ToASCII(volURI)
	if err != nil {
		return false
	}
	re := regexp.MustCompile(validVolURIRegex)
	return re.MatchString(volURI)
}

func parseVolURI(volURI string) string {
	volURI, _ = idna.ToASCII(volURI)
	re := regexp.MustCompile(validVolURIRegex)
	res := re.FindAllStringSubmatch(volURI, -1)
	volServers := strings.Split(res[0][1], ",")
	volumeID := res[0][10]
	subDir := res[0][11]

	if subDir == "" {
		return fmt.Sprintf("--volfile-id='%s' -s '%s'", volumeID, strings.Join(volServers, "' -s '"))
	}
	return fmt.Sprintf("--volfile-id='%s' --subdir-mount='%s' -s '%s'", volumeID, subDir, strings.Join(volServers, "' -s '"))
}

//GetMountName get moint point base on request and driver config (mountUniqName)
func GetMountName(d *basic.Driver, r *volume.CreateRequest) (string, error) {
	if r.Options == nil || r.Options["voluri"] == "" {
		return "", fmt.Errorf("voluri option required")
	}
	r.Options["voluri"] = strings.Trim(r.Options["voluri"], "\"")
	if !isValidURI(r.Options["voluri"]) {
		return "", fmt.Errorf("voluri option is malformated")
	}

	if d.Config.CustomOptions["mountUniqName"].(bool) {
		return url.PathEscape(r.Options["voluri"]), nil
	}
	return url.PathEscape(r.Name), nil
}
