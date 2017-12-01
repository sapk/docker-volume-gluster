package common

import (
	"fmt"
	"os"
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
	RunCmd(string) error
}

//Volume needed interface for some commons interactions
type Volume interface {
	GetMount() string
	GetRemote() string
	GetConnections() *int
}

//Mount needed interface for some commons interactions
type Mount interface {
	GetPath() string
	GetConnections() *int
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
		vols = append(vols, &volume.Volume{Name: name, Mountpoint: m.GetPath()})
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
	if *v.GetConnections() == 0 {
		if *m.GetConnections() == 0 {
			if err := os.Remove(m.GetPath()); err != nil {
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
	v, m, err := getVolumeMount(d, vName)
	if err == nil && v != nil && m != nil && *m.GetConnections() > 0 {
		*v.GetConnections()++
		*m.GetConnections()++
		return v, m, d.SaveConfig()
	}
	return v, m, err
}

//Unmount wrapper around github.com/docker/go-plugins-helpers/volume
func Unmount(d Driver, vName string) error {
	log.Debugf("Entering Unmount: name: %s", vName)
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	_, m, err := getVolumeMount(d, vName)
	if err != nil {
		return err
	}

	if *m.GetConnections() <= 1 {
		cmd := fmt.Sprintf("/usr/bin/umount %s", m.GetPath())
		if err := d.RunCmd(cmd); err != nil {
			return err
		}
		*m.GetConnections() = 0
		*m.GetConnections() = 0
	} else {
		*m.GetConnections()--
		*m.GetConnections()--
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
