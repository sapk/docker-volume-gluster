package common

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/go-plugins-helpers/volume"
)

//Driver needed interface for some commons interactions
type Driver interface {
	GetLock() *sync.RWMutex
	GetVolumes() map[string]Volume
	GetMounts() map[string]Mount
	SaveConfig() error
	StartCmd(name string, arg ...string) (*exec.Cmd, error)
}

//Volume needed interface for some commons interactions
type Volume interface {
	increasable
	GetMount() string
	GetRemote() string
	GetStatus() map[string]interface{}
}

//Mount needed interface for some commons interactions
type Mount interface {
	increasable
	GetPath() string
	//GetProcess() *exec.Cmd
	SetProcess(*exec.Cmd)
}

type increasable interface {
	GetConnections() int
	SetConnections(int)
}

func getMount(d Driver, mPath string) (Mount, error) {
	m, ok := d.GetMounts()[mPath]
	if !ok {
		return nil, fmt.Errorf("mount %s not found", mPath)
	}
	log.Debugf("Mount found: %s", m)
	return m, nil
}

func getVolumeMount(d Driver, vName string) (Volume, Mount, error) {
	v, ok := d.GetVolumes()[vName]
	if !ok {
		return nil, nil, fmt.Errorf("volume %s not found", vName)
	}
	log.Debugf("Volume found: %s", v)
	m, err := getMount(d, v.GetMount())
	return v, m, err
}

//List wrapper around github.com/docker/go-plugins-helpers/volume
func List(d Driver) (*volume.ListResponse, error) {
	log.Debugf("Entering List")
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	var vols []*volume.Volume
	for name, v := range d.GetVolumes() {
		log.Debugf("Volume found: %s", v)
		m, err := getMount(d, v.GetMount())
		if err != nil {
			return nil, err
		}
		vols = append(vols, &volume.Volume{Name: name, Status: v.GetStatus(), Mountpoint: m.GetPath()})
	}
	return &volume.ListResponse{Volumes: vols}, nil
}

//Get wrapper around github.com/docker/go-plugins-helpers/volume
func Get(d Driver, vName string) (Volume, Mount, error) {
	log.Debugf("Entering Get: name: %s", vName)
	d.GetLock().RLock()
	defer d.GetLock().RUnlock()
	return getVolumeMount(d, vName)
}

//Remove wrapper around github.com/docker/go-plugins-helpers/volume
func Remove(d Driver, vName string) error {
	log.Debugf("Entering Remove: name: %s", vName)
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	v, m, err := getVolumeMount(d, vName)
	if err != nil {
		return err
	}
	if v.GetConnections() == 0 {
		if m.GetConnections() == 0 {
			if err := os.Remove(m.GetPath()); err != nil && !strings.Contains(err.Error(), "no such file or directory") {
				return err
			}
			delete(d.GetMounts(), v.GetMount())
		}
		delete(d.GetVolumes(), vName)
		return d.SaveConfig()
	}
	return fmt.Errorf("volume %s is currently used by a container", vName)
}

//MountExist wrapper around github.com/docker/go-plugins-helpers/volume
func MountExist(d Driver, vName string) (Volume, Mount, error) {
	log.Debugf("Entering MountExist: name: %s", vName)
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	return getVolumeMount(d, vName)
}

func SetN(val int, oList ...increasable) {
	for _, o := range oList {
		o.SetConnections(val)
	}
}

func AddN(val int, oList ...increasable) {
	for _, o := range oList {
		o.SetConnections(o.GetConnections() + val)
	}
}

//Unmount wrapper around github.com/docker/go-plugins-helpers/volume
func Unmount(d Driver, vName string) error {
	log.Debugf("Entering Unmount: name: %s", vName)
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	v, m, err := getVolumeMount(d, vName)
	if err != nil {
		return err
	}

	if m.GetConnections() <= 1 {
		c, err := d.StartCmd("/usr/bin/umount", m.GetPath())
		if err != nil {
			return err
		}
		err = c.Wait()
		if err != nil {
			return err
		}
		SetN(0, m, v)
	} else {
		AddN(-1, m, v)
	}

	return d.SaveConfig()
}

//Capabilities wrapper around github.com/docker/go-plugins-helpers/volume
func Capabilities() *volume.CapabilitiesResponse {
	log.Debugf("Entering Capabilities")
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}
