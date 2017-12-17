package driver

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sapk/docker-volume-gluster/common"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
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

type GlusterMountpoint struct {
	Path        string    `json:"path"`
	Connections int       `json:"connections"`
	Process     *exec.Cmd `json:"-"`
}

func (d *GlusterMountpoint) GetPath() string {
	return d.Path
}
func (d *GlusterMountpoint) SetProcess(c *exec.Cmd) {
	d.Process = c
}

func (d *GlusterMountpoint) GetConnections() int {
	return d.Connections
}

func (d *GlusterMountpoint) SetConnections(n int) {
	d.Connections = n
}

type GlusterVolume struct {
	VolumeURI   string `json:"voluri"`
	Mount       string `json:"mount"`
	Connections int    `json:"connections"`
}

func (v *GlusterVolume) GetMount() string {
	return v.Mount
}

func (v *GlusterVolume) GetRemote() string {
	return v.VolumeURI
}

func (v *GlusterVolume) GetConnections() int {
	return v.Connections
}

func (v *GlusterVolume) SetConnections(n int) {
	v.Connections = n
}

func (v *GlusterVolume) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"TODO": "List",
	}
}

//GlusterDriver the global driver responding to call
type GlusterDriver struct {
	lock          sync.RWMutex
	root          string
	binary        string
	mountUniqName bool
	persitence    *viper.Viper
	volumes       map[string]*GlusterVolume
	mounts        map[string]*GlusterMountpoint
	logOut        *io.PipeWriter
	logErr        *io.PipeWriter
}

func (d *GlusterDriver) GetVolumes() map[string]common.Volume {
	vi := make(map[string]common.Volume, len(d.volumes))
	for k, i := range d.volumes {
		vi[k] = i
	}
	return vi
}

func (d *GlusterDriver) GetMounts() map[string]common.Mount {
	mi := make(map[string]common.Mount, len(d.mounts))
	for k, i := range d.mounts {
		mi[k] = i
	}
	return mi
}

func (d *GlusterDriver) GetLock() *sync.RWMutex {
	return &d.lock
}

//Init start all needed deps and serve response to API call
func Init(root string, mountUniqName bool) *GlusterDriver {
	log.Debugf("Init gluster driver at %s, UniqName: %v", root, mountUniqName)
	logger := log.New() //TODO defer close writer
	path, err := exec.LookPath("glusterfs")
	if err != nil {
		log.Fatal("glusterfs binary not found")
	}

	d := &GlusterDriver{
		root:          root,
		binary:        path,
		mountUniqName: mountUniqName,
		persitence:    viper.New(),
		volumes:       make(map[string]*GlusterVolume),
		mounts:        make(map[string]*GlusterMountpoint),
		logOut:        logger.WriterLevel(log.DebugLevel),
		logErr:        logger.WriterLevel(log.ErrorLevel),
	}

	d.persitence.SetDefault("volumes", map[string]*GlusterVolume{})
	d.persitence.SetConfigName("persistence")
	d.persitence.SetConfigType("json")
	d.persitence.AddConfigPath(CfgFolder)
	if err := d.persitence.ReadInConfig(); err != nil { // Handle errors reading the config file
		log.Warn("No persistence file found, I will start with a empty list of volume.", err)
	} else {
		log.Debug("Retrieving volume list from persistence file.")

		var version int
		err := d.persitence.UnmarshalKey("version", &version)
		if err != nil || version != CfgVersion {
			log.Warn("Unable to decode version of persistence, %v", err)
			d.volumes = make(map[string]*GlusterVolume)
			d.mounts = make(map[string]*GlusterMountpoint)
		} else { //We have the same version
			err := d.persitence.UnmarshalKey("volumes", &d.volumes)
			if err != nil {
				log.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.volumes = make(map[string]*GlusterVolume)
			}
			err = d.persitence.UnmarshalKey("mounts", &d.mounts)
			if err != nil {
				log.Warn("Unable to decode into struct -> start with empty list, %v", err)
				d.mounts = make(map[string]*GlusterMountpoint)
			}
		}
	}
	return d
}

//Create create and init the requested volume
func (d *GlusterDriver) Create(r *volume.CreateRequest) error {
	log.Debugf("Entering Create: name: %s, options %v", r.Name, r.Options)

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
			return fmt.Errorf("%v already exist and is not empty !", m.Path)
		}
		d.mounts[v.Mount] = m
	}

	d.volumes[r.Name] = v
	log.Debugf("Volume Created: %v", v)
	if err := d.SaveConfig(); err != nil {
		return err
	}
	return nil
}

//List volumes handled by these driver
func (d *GlusterDriver) List() (*volume.ListResponse, error) {
	return common.List(d)
}

//Get get info on the requested volume
func (d *GlusterDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	v, m, err := common.Get(d, r.Name)
	if err != nil {
		return nil, err
	}
	return &volume.GetResponse{Volume: &volume.Volume{Name: r.Name, Status: v.GetStatus(), Mountpoint: m.GetPath()}}, nil
}

//Remove remove the requested volume
func (d *GlusterDriver) Remove(r *volume.RemoveRequest) error {
	return common.Remove(d, r.Name)
}

//Path get path of the requested volume
func (d *GlusterDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	_, m, err := common.Get(d, r.Name)
	if err != nil {
		return nil, err
	}
	return &volume.PathResponse{Mountpoint: m.GetPath()}, nil
}

//Mount mount the requested volume
func (d *GlusterDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	log.Debugf("Entering Mount: %v", r)

	v, m, err := common.MountExist(d, r.Name)
	if err != nil {
		return nil, err
	}
	if m != nil && m.GetConnections() > 0 {
		return &volume.MountResponse{Mountpoint: m.GetPath()}, nil
	}

	d.GetLock().Lock()
	defer d.GetLock().Unlock()

	//c, err := d.StartCmd(d.binary, append(parseVolURI(v.GetRemote()), "--log-level", "INFO", "--log-file", "/dev/stdout", "--no-daemon", m.GetPath())...) // TODO --debug
	c, err := d.StartCmd(d.binary, append(parseVolURI(v.GetRemote()), "--no-daemon", m.GetPath())...) // TODO --debug

	if err != nil {
		return nil, err
	}

	done := make(chan error, 1)
	go func() {
		done <- c.Wait()
	}()
	// Wait if failed for 5 seconds
	select {
	case err := <-done:
		return nil, err
	case <-time.After(time.Second * 5):
		m.SetProcess(c)
		common.AddN(1, v, m)
		return &volume.MountResponse{Mountpoint: m.GetPath()}, d.SaveConfig()
	}
}

//Unmount unmount the requested volume
func (d *GlusterDriver) Unmount(r *volume.UnmountRequest) error {
	return common.Unmount(d, r.Name)
}

//Capabilities Send capabilities of the local driver
func (d *GlusterDriver) Capabilities() *volume.CapabilitiesResponse {
	return common.Capabilities()
}
