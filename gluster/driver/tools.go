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
	validVolUriRegex = `([^:]+?):\/?([^\/]+)(/.+)?`
)

func isValidURI(volURI string) bool {
	re := regexp.MustCompile(validVolUriRegex)
	return re.MatchString(volURI)
}

func parseVolURI(volURI string) string {
	re := regexp.MustCompile(validVolUriRegex)
	res := re.FindAllStringSubmatch(volParts[1], -1)
	volServers := strings.Split(res[0][1], ",")
	volumeId := res[0][2]
	subDir := res[0][3]
	
	if (subDir == "") {
		return fmt.Sprintf("--volfile-id='%s' -s '%s'", volumeId, strings.Join(volServers, "' -s '"))
	} else {
		return fmt.Sprintf("--volfile-id='%s' --subdir-mount='%s' -s '%s'", volumeId, subDir, strings.Join(volServers, "' -s '"))
	}
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
