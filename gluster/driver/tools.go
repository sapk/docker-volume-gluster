package driver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	log "github.com/Sirupsen/logrus"
)

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
	b, err := json.Marshal(GlusterPersistence{Volumes: d.volumes})
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
