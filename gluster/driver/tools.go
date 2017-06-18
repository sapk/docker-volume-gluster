package driver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
)

//GlusterPersistence represent struct of persistence file
type GlusterPersistence struct {
	Version int                           `json:"version"`
	Volumes map[string]*glusterVolume     `json:"volumes"`
	Mounts  map[string]*glusterMountpoint `json:"mounts"`
}

func (d *GlusterDriver) saveConfig() error {
	fi, err := os.Lstat(cfgFolder)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(cfgFolder, 0700); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if fi != nil && !fi.IsDir() {
		return fmt.Errorf("%v already exist and it's not a directory", d.root)
	}
	b, err := json.Marshal(GlusterPersistence{Version: cfgVersion, Volumes: d.volumes, Mounts: d.mounts})
	if err != nil {
		log.Warn("Unable to encode persistence struct, %v", err)
	}
	//log.Debug("Writing persistence struct, %v", b, d.volumes)
	err = ioutil.WriteFile(cfgFolder+"/persistence.json", b, 0600)
	if err != nil {
		log.Warn("Unable to write persistence struct, %v", err)
	}
	//TODO display error messages
	return err
}

// run deamon in context of this gvfs drive with custome env
func (d *GlusterDriver) runCmd(cmd string) error {
	log.Debugf(cmd)
	return exec.Command("sh", "-c", cmd).Run()
}

func getMountName(d *GlusterDriver, r volume.Request) string {
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
