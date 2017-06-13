package driver

import (
	"fmt"
	"os"
	"io"
	"path/filepath"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/spf13/viper"
)

const (
	//MountTimeout timeout before killing a mount try in seconds
	MountTimeout = 30
	cfgFolder    = "/etc/docker-volumes/gluster/"
)

type glusterVolume struct {
	VolumeURI   string `json:"voluri"`
	Mountpoint  string `json:"mountpoint,omitempty"`
	connections int
}

//GlusterDriver the global driver responding to call
type GlusterDriver struct {
	sync.RWMutex
	root       string
	fuseOpts   string
	persitence *viper.Viper
	volumes    map[string]*glusterVolume
}

//GlusterPersistence represent struct of persistence file
type GlusterPersistence struct {
	Volumes map[string]*glusterVolume `json:"volumes"`
}

//Init start all needed deps and serve response to API call
func Init(root string, fuseOpts string) *GlusterDriver {
	d := &GlusterDriver{
		root:       root,
		fuseOpts:   fuseOpts,
		persitence: viper.New(),
		volumes:    make(map[string]*glusterVolume),
	}
	d.persitence.SetDefault("volumes", map[string]*glusterVolume{})
	d.persitence.SetConfigName("gluster-persistence")
	d.persitence.SetConfigType("json")
	d.persitence.AddConfigPath(cfgFolder)
	if err := d.persitence.ReadInConfig(); err != nil { // Handle errors reading the config file
		log.Warn("No persistence file found, I will start with a empty list of volume.", err)
	} else {
		log.Debug("Retrieving volume list from persistence file.")
		/**/
		err := d.persitence.UnmarshalKey("volumes", &d.volumes)
		if err != nil {
			log.Warn("Unable to decode into struct -> start with empty list, %v", err)
			d.volumes = make(map[string]*glusterVolume)
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
		Mountpoint:  filepath.Join(d.root, r.Name),
		connections: 0,
	}

	_, err := os.Lstat(v.Mountpoint) //Create folder if not exist. This will also failed if already exist
	if os.IsNotExist(err) {
		if err = os.MkdirAll(v.Mountpoint, 0700); err != nil {
			return volume.Response{Err: err.Error()}
		}
	} else if err != nil {
		return volume.Response{Err: err.Error()}
	}
	
	isempty, err := isEmpty(v.Mountpoint)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	if isempty {
		d.volumes[r.Name] = v
		log.Debugf("Volume Created: %v", v)
		if err = d.saveConfig(); err != nil {
			return volume.Response{Err: err.Error()}
		}
		return volume.Response{}
	}
	
	return volume.Response{Err: fmt.Sprintf("%v already exist and is not empty !", v.Mountpoint)}
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

//Remove remove the requested volume
func (d *GlusterDriver) Remove(r volume.Request) volume.Response {
	log.Debugf("Entering Remove: name: %s, options %v", r.Name, r.Options)
	d.Lock()
	defer d.Unlock()
	v, ok := d.volumes[r.Name]

	if !ok {
		return volume.Response{Err: fmt.Sprintf("volume %s not found", r.Name)}
	}
	if v.connections == 0 {
		delete(d.volumes, r.Name)
		if err := os.Remove(v.Mountpoint); err != nil {
			return volume.Response{Err: err.Error()}
		}
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
		vols = append(vols, &volume.Volume{Name: name, Mountpoint: v.Mountpoint})
		log.Debugf("Volume found: %s", v)
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
	return volume.Response{Volume: &volume.Volume{Name: r.Name, Mountpoint: v.Mountpoint}}
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
	return volume.Response{Mountpoint: v.Mountpoint}
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

	if v.connections > 0 {
		v.connections++
		return volume.Response{Mountpoint: v.Mountpoint}
	}

	cmd := fmt.Sprintf("/usr/bin/mount -t glusterfs %s %s", v.VolumeURI, v.Mountpoint) //TODO fuseOpts   /usr/bin/mount -t glusterfs v.VolumeURI -o fuseOpts v.Mountpoint
	if err := d.runCmd(cmd); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if err := d.saveConfig(); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{Mountpoint: v.Mountpoint}
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

	if v.connections <= 1 {
		cmd := fmt.Sprintf("/usr/bin/umount %s", v.Mountpoint)
		if err := d.runCmd(cmd); err != nil {
			return volume.Response{Err: err.Error()}
		}
		v.connections = 0
	} else {
		v.connections--
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
