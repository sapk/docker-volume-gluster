package driver

import (
	"fmt"

	"github.com/sapk/docker-volume-helpers/basic"
	"github.com/sapk/docker-volume-helpers/driver"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	//MountTimeout timeout before killing a mount try in seconds
	MountTimeout = 30
	//CfgVersion current config version compat
	CfgVersion = 1
	//CfgFolder config folder
	CfgFolder = "/etc/docker-volumes/gluster/"
)

type GlusterDriver = basic.Driver

//Init start all needed deps and serve response to API call
func Init(root string, mountUniqName bool) *GlusterDriver {
	logrus.Debugf("Init gluster driver at %s, UniqName: %v", root, mountUniqName)
	d := &GlusterDriver{
		Root:          root,
		MountUniqName: mountUniqName,
		Persistence:   viper.New(),
		Volumes:       make(map[string]*basic.Volume),
		Mounts:        make(map[string]*basic.Mountpoint),
		CfgFolder:     CfgFolder,
		Version:       CfgVersion,
		IsValidURI:    isValidURI,
		MountVolume: func(d *basic.Driver, v driver.Volume, m driver.Mount, r *volume.MountRequest) (*volume.MountResponse, error) {
			cmd := fmt.Sprintf("glusterfs %s %s", parseVolURI(v.GetRemote()), m.GetPath())
			//cmd := fmt.Sprintf("/usr/bin/mount -t glusterfs %s %s", v.VolumeURI, m.Path)
			//TODO fuseOpts   /usr/bin/mount -t glusterfs v.VolumeURI -o fuseOpts v.Mountpoint
			if err := d.RunCmd(cmd); err != nil {
				return nil, err
			}
			return &volume.MountResponse{Mountpoint: m.GetPath()}, nil
		},
	}

	d.Persistence.SetDefault("volumes", map[string]*basic.Volume{})
	d.Persistence.SetConfigName("persistence")
	d.Persistence.SetConfigType("json")
	d.Persistence.AddConfigPath(CfgFolder)
	if err := d.Persistence.ReadInConfig(); err != nil { // Handle errors reading the config file
		logrus.Warn("No persistence file found, I will start with a empty list of volume.", err)
	} else {
		logrus.Debug("Retrieving volume list from persistence file.")

		var version int
		err := d.Persistence.UnmarshalKey("version", &version)
		if err != nil || version != CfgVersion {
			logrus.Warn("Unable to decode version of persistence, %v", err)
			d.Volumes = make(map[string]*basic.Volume)
			d.Mounts = make(map[string]*basic.Mountpoint)
		} else { //We have the same version
			err := d.Persistence.UnmarshalKey("volumes", &d.Volumes)
			if err != nil {
				logrus.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.Volumes = make(map[string]*basic.Volume)
			}
			err = d.Persistence.UnmarshalKey("mounts", &d.Mounts)
			if err != nil {
				logrus.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.Mounts = make(map[string]*basic.Mountpoint)
			}
		}
	}
	return d
}
