package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

//Init start all needed deps and serve response to API call
func Init(root string, mountUniqName bool) *GlusterDriver {
	logrus.Debugf("Init gluster driver at %s, UniqName: %v", root, mountUniqName)
	d := &GlusterDriver{
		root:          root,
		mountUniqName: mountUniqName,
		persitence:    viper.New(),
		volumes:       make(map[string]*GlusterVolume),
		mounts:        make(map[string]*GlusterMountpoint),
	}

	d.persitence.SetDefault("volumes", map[string]*GlusterVolume{})
	d.persitence.SetConfigName("persistence")
	d.persitence.SetConfigType("json")
	d.persitence.AddConfigPath(CfgFolder)
	if err := d.persitence.ReadInConfig(); err != nil { // Handle errors reading the config file
		logrus.Warn("No persistence file found, I will start with a empty list of volume.", err)
	} else {
		logrus.Debug("Retrieving volume list from persistence file.")

		var version int
		err := d.persitence.UnmarshalKey("version", &version)
		if err != nil || version != CfgVersion {
			logrus.Warn("Unable to decode version of persistence, %v", err)
			d.volumes = make(map[string]*GlusterVolume)
			d.mounts = make(map[string]*GlusterMountpoint)
		} else { //We have the same version
			err := d.persitence.UnmarshalKey("volumes", &d.volumes)
			if err != nil {
				logrus.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.volumes = make(map[string]*GlusterVolume)
			}
			err = d.persitence.UnmarshalKey("mounts", &d.mounts)
			if err != nil {
				logrus.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.mounts = make(map[string]*GlusterMountpoint)
			}
		}
	}
	return d
}

//Create create and init the requested volume
func (d *GlusterDriver) Create(r *volume.CreateRequest) error {
	logrus.Debugf("Entering Create: name: %s, options %v", r.Name, r.Options)

	if r.Options == nil || r.Options["voluri"] == "" {
		return fmt.Errorf("voluri option required")
	}
	r.Options["voluri"] = strings.Trim(r.Options["voluri"], "\"")
	if !isValidURI(r.Options["voluri"]) {
		return fmt.Errorf("voluri option is malformated")
	}

	d.GetLock().Lock()
	defer d.GetLock().Unlock()

	v := &GlusterVolume{
		VolumeURI:   r.Options["voluri"],
		Mount:       getMountName(d, r),
		Connections: 0,
	}

	if _, ok := d.mounts[v.Mount]; !ok { //This mountpoint doesn't allready exist -> create it
		m := &GlusterMountpoint{
			Path:        filepath.Join(d.root, v.Mount),
			Connections: 0,
		}

		_, err := os.Lstat(m.Path) //Create folder if not exist. This will also failed if already exist
		if os.IsNotExist(err) {
			if err = os.MkdirAll(m.Path, 0700); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		isempty, err := isEmpty(m.Path)
		if err != nil {
			return err
		}
		if !isempty {
			return fmt.Errorf("%v already exist and is not empty", m.Path)
		}
		d.mounts[v.Mount] = m
	}

	d.volumes[r.Name] = v
	logrus.Debugf("Volume Created: %v", v)
	return d.SaveConfig()
}

//Path get path of the requested volume
func (d *GlusterDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	_, m, err := driver.Get(d, r.Name)
	if err != nil {
		return nil, err
	}
	return &volume.PathResponse{Mountpoint: m.GetPath()}, nil
}

//Mount mount the requested volume
func (d *GlusterDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	logrus.Debugf("Entering Mount: %v", r)

	v, m, err := driver.MountExist(d, r.Name)
	if err != nil {
		return nil, err
	}
	if m != nil && m.GetConnections() > 0 {
		return &volume.MountResponse{Mountpoint: m.GetPath()}, nil
	}

	d.GetLock().Lock()
	defer d.GetLock().Unlock()

	cmd := fmt.Sprintf("glusterfs %s %s", parseVolURI(v.GetRemote()), m.GetPath())
	//cmd := fmt.Sprintf("/usr/bin/mount -t glusterfs %s %s", v.VolumeURI, m.Path)
	//TODO fuseOpts   /usr/bin/mount -t glusterfs v.VolumeURI -o fuseOpts v.Mountpoint
	if err := d.RunCmd(cmd); err != nil {
		return nil, err
	}
	//time.Sleep(3 * time.Second)
	driver.AddN(1, v, m)
	return &volume.MountResponse{Mountpoint: m.GetPath()}, d.SaveConfig()
}
