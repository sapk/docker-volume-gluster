package driver

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/rs/zerolog/log"
)

//Driver needed interface for some commons interactions
type Driver interface {
	GetLock() *sync.RWMutex
	GetVolumes() map[string]Volume
	RemoveVolume(string) error
	GetMounts() map[string]Mount
	RemoveMount(string) error
	SaveConfig() error
	RunCmd(string) error
}

//Volume needed interface for some commons interactions
type Volume interface {
	increasable
	GetMount() string
	GetOptions() map[string]string
	GetStatus() map[string]interface{}
}

//Mount needed interface for some commons interactions
type Mount interface {
	increasable
	GetPath() string
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
	log.Debug().Msgf("Mount found: %s", m)
	return m, nil
}

func getVolumeMount(d Driver, vName string) (Volume, Mount, error) {
	v, ok := d.GetVolumes()[vName]
	if !ok {
		return nil, nil, fmt.Errorf("volume %s not found", vName)
	}
	log.Debug().Msgf("Volume found: %s", v)
	m, err := getMount(d, v.GetMount())
	return v, m, err
}

//List wrapper around github.com/docker/go-plugins-helpers/volume
func List(d Driver) (*volume.ListResponse, error) {
	log.Debug().Msgf("Entering List")
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	var vols []*volume.Volume
	for name, v := range d.GetVolumes() {
		log.Debug().Msgf("Volume found: %s", v)
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
	log.Debug().Msgf("Entering Get: name: %s", vName)
	d.GetLock().RLock()
	defer d.GetLock().RUnlock()
	return getVolumeMount(d, vName)
}

//Remove wrapper around github.com/docker/go-plugins-helpers/volume
func Remove(d Driver, vName string) error {
	log.Debug().Msgf("Entering Remove: name: %s", vName)
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
			if err := d.RemoveMount(v.GetMount()); err != nil {
				return err
			}
		}
		if err := d.RemoveVolume(vName); err != nil {
			return err
		}
		return d.SaveConfig()
	}
	return fmt.Errorf("volume %s is currently used by a container", vName)
}

//MountExist wrapper around github.com/docker/go-plugins-helpers/volume
func MountExist(d Driver, vName string) (Volume, Mount, error) {
	log.Debug().Msgf("Entering MountExist: name: %s", vName)
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	return getVolumeMount(d, vName)
}

//SetN set the value on a increasable
func SetN(val int, oList ...increasable) {
	for _, o := range oList {
		o.SetConnections(val)
	}
}

//AddN add the value on a increasable
func AddN(val int, oList ...increasable) {
	for _, o := range oList {
		o.SetConnections(o.GetConnections() + val)
	}
}

//Unmount wrapper around github.com/docker/go-plugins-helpers/volume
func Unmount(d Driver, vName string) error {
	log.Debug().Msgf("Entering Unmount: name: %s", vName)
	d.GetLock().Lock()
	defer d.GetLock().Unlock()
	v, m, err := getVolumeMount(d, vName)
	if err != nil {
		return err
	}

	if m.GetConnections() <= 1 {
		cmd := fmt.Sprintf("/usr/bin/umount %s", m.GetPath())
		if err := d.RunCmd(cmd); err != nil {
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
	log.Debug().Msgf("Entering Capabilities")
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}
