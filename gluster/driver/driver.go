package driver

import (
	"fmt"

	"github.com/sapk/docker-volume-helpers/basic"
	"github.com/sapk/docker-volume-helpers/driver"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
)

var (
	//MountTimeout timeout before killing a mount try in seconds
	MountTimeout = 30
	//CfgVersion current config version compat
	CfgVersion = 1
	//CfgFolder config folder
	CfgFolder = "/etc/docker-volumes/gluster/"
)

//GlusterDriver docker volume plugin driver extension of basic plugin
type GlusterDriver = basic.Driver

//Init start all needed deps and serve response to API call
func Init(root string, mountUniqName bool) *GlusterDriver {
	logrus.Debugf("Init gluster driver at %s, UniqName: %v", root, mountUniqName)
	config := basic.DriverConfig{
		Version:       CfgVersion,
		Root:          root,
		Folder:        CfgFolder,
		MountUniqName: mountUniqName,
	}
	eventHandler := basic.DriverEventHandler{
		IsValidURI:    isValidURI,
		OnMountVolume: mountVolume,
	}
	return basic.Init(&config, &eventHandler)
}

func mountVolume(d *basic.Driver, v driver.Volume, m driver.Mount, r *volume.MountRequest) (*volume.MountResponse, error) {
	cmd := fmt.Sprintf("glusterfs %s %s", parseVolURI(v.GetRemote()), m.GetPath())
	//cmd := fmt.Sprintf("/usr/bin/mount -t glusterfs %s %s", v.VolumeURI, m.Path)
	//TODO fuseOpts   /usr/bin/mount -t glusterfs v.VolumeURI -o fuseOpts v.Mountpoint
	if err := d.RunCmd(cmd); err != nil {
		return nil, err
	}
	return &volume.MountResponse{Mountpoint: m.GetPath()}, nil
}
