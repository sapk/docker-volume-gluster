package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
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
		logrus.Warn("Unable to encode persistence struct, %v", err)
	}
	//logrus.Debug("Writing persistence struct, %v", b, d.volumes)
	err = ioutil.WriteFile(CfgFolder+"/persistence.json", b, 0600)
	if err != nil {
		logrus.Warn("Unable to write persistence struct, %v", err)
		return fmt.Errorf("SaveConfig: %s", err)
	}
	return nil
}

//RunCmd run deamon in context of this gvfs drive with custome env
func (d *GlusterDriver) RunCmd(cmd string) error {
	logrus.Debugf(cmd)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		logrus.Debugf("Error: %v", err)
	}
	logrus.Debugf("Output: %v", out)
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

//based on: http://stackoverflow.com/questions/30697324/how-to-check-if-directory-on-path-is-empty
func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}
