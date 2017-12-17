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
	log "github.com/sirupsen/logrus"
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
		log.Warn("Unable to encode persistence struct, %v", err)
	}
	//log.Debug("Writing persistence struct, %v", b, d.volumes)
	err = ioutil.WriteFile(CfgFolder+"/persistence.json", b, 0600)
	if err != nil {
		log.Warn("Unable to write persistence struct, %v", err)
		return fmt.Errorf("SaveConfig: %s", err)
	}
	return nil
}

//StartCmd start deamon in context of this gluster driver and keep it in backgroud
func (d *GlusterDriver) StartCmd(bin string, arg ...string) (*exec.Cmd, error) {
	log.Debugf("%s %s", bin, strings.Join(arg, " "))
	c := exec.Command(bin, arg...)
	c.Stdout = d.logOut
	c.Stderr = d.logErr
	return c, c.Start()
}

func isValidURI(volURI string) bool {
	re := regexp.MustCompile(validHostnameRegex + ":.+")
	return re.MatchString(volURI)
}

func parseVolURI(volURI string) []string {
	volParts := strings.Split(volURI, ":")
	volServers := strings.Split(volParts[0], ",")
	ret := make([]string, 1+len(volServers))
	ret[0] = fmt.Sprintf("--volfile-id=%s", volParts[1])
	for i, server := range volServers {
		ret[i+1] = fmt.Sprintf("--volfile-server=%s", server)
	}
	return ret
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
