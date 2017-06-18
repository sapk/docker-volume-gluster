package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/spf13/viper"
)

const (
	//MountTimeout timeout before killing a mount try in seconds
	MountTimeout = 30
	cfgVersion   = 1
	cfgFolder    = "/etc/docker-volumes/gluster/"
)

type glusterMountpoint struct {
	Path        string `json:"path"`
	Connections int    `json:"connections"`
}

type glusterVolume struct {
	VolumeURI   string `json:"voluri"`
	Mount       string `json:"mount"`
	Connections int    `json:"connections"`
}

//GlusterDriver the global driver responding to call
type GlusterDriver struct {
	sync.RWMutex
	root          string
	fuseOpts      string
	mountUniqName bool
	persitence    *viper.Viper
	volumes       map[string]*glusterVolume
	mounts        map[string]*glusterMountpoint
}

//Init start all needed deps and serve response to API call
func Init(root string, fuseOpts string, mountUniqName bool) *GlusterDriver {
	d := &GlusterDriver{
		root:          root,
		fuseOpts:      fuseOpts,
		mountUniqName: mountUniqName,
		persitence:    viper.New(),
		volumes:       make(map[string]*glusterVolume),
		mounts:        make(map[string]*glusterMountpoint),
	}

	d.persitence.SetDefault("volumes", map[string]*glusterVolume{})
	d.persitence.SetConfigName("gluster-persistence")
	d.persitence.SetConfigType("json")
	d.persitence.AddConfigPath(cfgFolder)
	if err := d.persitence.ReadInConfig(); err != nil { // Handle errors reading the config file
		log.Warn("No persistence file found, I will start with a empty list of volume.", err)
	} else {
		log.Debug("Retrieving volume list from persistence file.")

		var version int
		err := d.persitence.UnmarshalKey("version", &version)
		if err != nil || version != cfgVersion {
			log.Warn("Unable to decode version of persistence, %v", err)
			d.volumes = make(map[string]*glusterVolume)
			d.mounts = make(map[string]*glusterMountpoint)
		} else { //We have the same version
			err := d.persitence.UnmarshalKey("volumes", &d.volumes)
			if err != nil {
				log.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.volumes = make(map[string]*glusterVolume)
			}
			err = d.persitence.UnmarshalKey("mounts", &d.mounts)
			if err != nil {
				log.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.mounts = make(map[string]*glusterMountpoint)
			}
		}
	}
	return d
}

//Create create and init the requested volume
func (d *GlusterDriver) Create(r volume.Request) volume.Response {
	log.Debugf("Entering Create: name: %s, options %v", r.Name, r.Options)
	d.Lock()
	defer d.Unlock()

	if r.Options == nil || r.Options["voluri"] == "" {
		return volume.Response{Err: "voluri option required"}
	}

	v := &glusterVolume{
		VolumeURI:   r.Options["voluri"],
		Mount:       getMountName(d, r),
		Connections: 0,
	}

	if _, ok := d.mounts[v.Mount]; !ok { //This mountpoint doesn't allready exist -> create it
		m := &glusterMountpoint{
			Path:        filepath.Join(d.root, v.Mount),
			Connections: 0,
		}

		_, err := os.Lstat(m.Path) //Create folder if not exist. This will also failed if already exist
		if os.IsNotExist(err) {
			if err = os.MkdirAll(m.Path, 0700); err != nil {
				return volume.Response{Err: err.Error()}
			}
		} else if err != nil {
			return volume.Response{Err: err.Error()}
		}
		isempty, err := isEmpty(m.Path)
		if err != nil {
			return volume.Response{Err: err.Error()}
		}
		if !isempty {
			return volume.Response{Err: fmt.Sprintf("%v already exist and is not empty !", m.Path)}
		}
		d.mounts[v.Mount] = m
	}

	d.volumes[r.Name] = v
	log.Debugf("Volume Created: %v", v)
	if err := d.saveConfig(); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}

}

//Remove remove the requested volume
func (d *GlusterDriver) Remove(r volume.Request) volume.Response {
	//TODO remove related mounts
	log.Debugf("Entering Remove: name: %s, options %v", r.Name, r.Options)
	d.Lock()
	defer d.Unlock()
	v, ok := d.volumes[r.Name]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume %s not found", r.Name)}
	}
	log.Debugf("Volume found: %s", v)

	m, ok := d.mounts[v.Mount]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume mount %s not found for %s", v.Mount, r.Name)}
	}
	log.Debugf("Mount found: %s", m)

	if v.Connections == 0 {
		if m.Connections == 0 {
			if err := os.Remove(m.Path); err != nil {
				return volume.Response{Err: err.Error()}
			}
			delete(d.mounts, v.Mount)
		}
		delete(d.volumes, r.Name)
		return volume.Response{}
	}
	if err := d.saveConfig(); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Err: fmt.Sprintf("volume %s is currently used by a container", r.Name)}
}

//List volumes handled by thos driver
func (d *GlusterDriver) List(r volume.Request) volume.Response {
	log.Debugf("Entering List: name: %s, options %v", r.Name, r.Options)
	d.Lock()
	defer d.Unlock()

	var vols []*volume.Volume
	for name, v := range d.volumes {
		log.Debugf("Volume found: %s", v)
		m, ok := d.mounts[v.Mount]
		if !ok {
			return volume.Response{Err: fmt.Sprintf("volume mount %s not found for %s", v.Mount, r.Name)}
		}
		log.Debugf("Mount found: %s", m)
		vols = append(vols, &volume.Volume{Name: name, Mountpoint: m.Path})
	}
	return volume.Response{Volumes: vols}
}

//Get get info on the requested volume
func (d *GlusterDriver) Get(r volume.Request) volume.Response {
	log.Debugf("Entering Get: name: %s", r.Name)
	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume %s not found", r.Name)}
	}
	log.Debugf("Volume found: %s", v)

	m, ok := d.mounts[v.Mount]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume mount %s not found for %s", v.Mount, r.Name)}
	}
	log.Debugf("Mount found: %s", m)

	return volume.Response{Volume: &volume.Volume{Name: r.Name, Mountpoint: m.Path}}
}

//Path get path of the requested volume
func (d *GlusterDriver) Path(r volume.Request) volume.Response {
	log.Debugf("Entering Path: name: %s, options %v", r.Name)
	d.RLock()
	defer d.RUnlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume %s not found", r.Name)}
	}
	log.Debugf("Volume found: %s", v)

	m, ok := d.mounts[v.Mount]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume mount %s not found for %s", v.Mount, r.Name)}
	}
	log.Debugf("Mount found: %s", m)

	return volume.Response{Mountpoint: m.Path}
}

//Mount mount the requested volume
func (d *GlusterDriver) Mount(r volume.MountRequest) volume.Response {
	log.Debugf("Entering Mount: %v", r)
	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume %s not found", r.Name)}
	}

	m, ok := d.mounts[v.Mount]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume mount %s not found for %s", v.Mount, r.Name)}
	}

	if m.Connections > 0 {
		v.Connections++
		m.Connections++
		if err := d.saveConfig(); err != nil {
			return volume.Response{Err: err.Error()}
		}
		return volume.Response{Mountpoint: m.Path}
	}

	cmd := fmt.Sprintf("/usr/bin/mount -t glusterfs %s %s", v.VolumeURI, m.Path) //TODO fuseOpts   /usr/bin/mount -t glusterfs v.VolumeURI -o fuseOpts v.Mountpoint
	if err := d.runCmd(cmd); err != nil {
		return volume.Response{Err: err.Error()}
	}

	v.Connections++
	m.Connections++
	if err := d.saveConfig(); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Mountpoint: m.Path}
}

//Unmount unmount the requested volume
func (d *GlusterDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.Debugf("Entering Unmount: %v", r)
	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume %s not found", r.Name)}
	}

	m, ok := d.mounts[v.Mount]
	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume mount %s not found for %s", v.Mount, r.Name)}
	}

	if m.Connections <= 1 {
		cmd := fmt.Sprintf("/usr/bin/umount %s", m.Path)
		if err := d.runCmd(cmd); err != nil {
			return volume.Response{Err: err.Error()}
		}
		m.Connections = 0
		v.Connections = 0
	} else {
		m.Connections--
		v.Connections--
	}

	if err := d.saveConfig(); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

//Capabilities Send capabilities of the local driver
func (d *GlusterDriver) Capabilities(r volume.Request) volume.Response {
	log.Debugf("Entering Capabilities: %v", r)
	return volume.Response{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}
