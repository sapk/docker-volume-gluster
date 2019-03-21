package driver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/rs/zerolog/log"
)

const (
	validHostnameRegex = `(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])`
)

//GlusterPersistence represent struct of persistence file
type GlusterPersistence struct {
	Version int                           `json:"version"`
	Volumes map[string]*GlusterVolume     `json:"volumes"`
	Mounts  map[string]*GlusterMountpoint `json:"mounts"`
}

//SaveConfig stroe config/state in file  //TODO put inside common
func (d *GlusterDriver) SaveConfig() error {
	fi, err := os.Lstat(CfgFolder)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(CfgFolder, 0700); err != nil {
			return fmt.Errorf("SaveConfig: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("SaveConfig: %s", err)
	}
	if fi != nil && !fi.IsDir() {
		return fmt.Errorf("SaveConfig: %v already exist and it's not a directory", d.root)
	}
	b, err := json.Marshal(GlusterPersistence{Version: CfgVersion, Volumes: d.volumes, Mounts: d.mounts})
	if err != nil {
		log.Warn().Msgf("Unable to encode persistence struct, %v", err)
	}
	//log.Debug("Writing persistence struct, %v", b, d.volumes)
	err = ioutil.WriteFile(CfgFolder+"/persistence.json", b, 0600)
	if err != nil {
		log.Warn().Msgf("Unable to write persistence struct, %v", err)
		return fmt.Errorf("SaveConfig: %s", err)
	}
	return nil
}

//RunCmd run deamon in context of this gvfs drive with custome env
func (d *GlusterDriver) RunCmd(cmd string) error {
	log.Debug().Msg(cmd)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		log.Debug().Msgf("Error: %v", err)
	}
	log.Debug().Msgf("Output: %v", out)
	return err
}

func isValidURI(volURI string) bool {
	re := regexp.MustCompile(validHostnameRegex + ":.+")
	return re.MatchString(volURI)
}

func parseVolURI(volURI string) string {
	volParts := strings.Split(volURI, ":")
	volServers := strings.Split(volParts[0], ",")
	return fmt.Sprintf("--volfile-id='%s' -s '%s'", volParts[1], strings.Join(volServers, "' -s '"))
}

func getMountName(d *GlusterDriver, r *volume.CreateRequest) string {
	if d.mountUniqName {
		return url.PathEscape(r.Options["voluri"])
	}
	return url.PathEscape(r.Name)
}
