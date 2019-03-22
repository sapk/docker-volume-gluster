package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/rs/zerolog/log"
)

//Driver needed interface for some commons interactions
type Driver interface {
	sync.Locker
	GetVolumes() map[string]Volume
	GetMounts() map[string]Mount
	SaveConfig() error
	RunCmd(string) error
}

//Volume needed interface for some commons interactions
type Volume interface {
	increasable
	GetMount() string
	GetRemote() string
	GetStatus() map[string]interface{}
	GetCreatedAt() string
}

//Mount needed interface for some commons interactions
type Mount interface {
	increasable
	GetPath() string
}

//IsMounted check if a mount is in /proc/mounts
func IsMounted(m Mount) (bool, error) {
	//TODO Better check for remote /var/lib/docker-volumes/rclone/mountpath fuse.rclone ro,nosuid,nodev,relatime,user_id=0,group_id=0 0 0
	buf, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return false, err
	}
	//log.Debug().Msgf("isMounted Path: path: %s %v", m.GetPath(), strings.Contains(string(buf), " "+m.GetPath()+" fuse.rclone"))
	//return strings.Contains(string(buf), " "+m.GetPath()+" fuse.rclone"), nil
	return strings.Contains(string(buf), " "+m.GetPath()+" fuse"), nil
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
	d.Lock()
	defer d.Unlock()

	var vols []*volume.Volume
	for name, v := range d.GetVolumes() {
		log.Debug().Msgf("Volume found: %s", v)
		m, err := getMount(d, v.GetMount())
		if err != nil {
			return nil, err
		}
		vols = append(vols, &volume.Volume{Name: name, Status: v.GetStatus(), Mountpoint: m.GetPath(), CreatedAt: v.GetCreatedAt()})
	}
	return &volume.ListResponse{Volumes: vols}, nil
}

//Get wrapper around github.com/docker/go-plugins-helpers/volume
func Get(d Driver, vName string) (Volume, Mount, error) {
	log.Debug().Msgf("Entering Get: name: %s", vName)
	d.Lock()
	defer d.Unlock()
	return getVolumeMount(d, vName)
}

//Remove wrapper around github.com/docker/go-plugins-helpers/volume
func Remove(d Driver, vName string) error {
	log.Debug().Msgf("Entering Remove: name: %s", vName)
	d.Lock()
	defer d.Unlock()
	v, m, err := getVolumeMount(d, vName)
	if err != nil {
		return err
	}

	//Unmount if needed
	mounted, err := IsMounted(m)
	if err != nil {
		return err
	}

	if v.GetConnections() == 0 {
		if m.GetConnections() == 0 {
			if mounted { //Only if mounted
				if err := d.RunCmd(fmt.Sprintf("umount \"%s\"", m.GetPath())); err != nil {
					return err
				}
			}
			if _, err := os.Stat(m.GetPath()); !os.IsNotExist(err) {
				//Remove mount point
				if err := os.Remove(m.GetPath()); err != nil {
					return err
				}
			}
			delete(d.GetMounts(), v.GetMount())
		}
		delete(d.GetVolumes(), vName)
	}
	return d.SaveConfig()
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
	log.Debug().Msgf("Entering Unmount: name: %s", vName)
	d.Lock()
	defer d.Unlock()
	v, m, err := getVolumeMount(d, vName)
	if err != nil {
		return err
	}

	mounted, err := IsMounted(m)
	if err != nil {
		return err
	}
	if !mounted { //Force reset if not mounted
		SetN(0, v, m)
	} else {
		if m.GetConnections() <= 1 {
			cmd := fmt.Sprintf("umount %s", m.GetPath())
			if err := d.RunCmd(cmd); err != nil {
				return err
			}
			SetN(0, m, v)
		} else {
			AddN(-1, m, v)
		}
	}
	return d.SaveConfig()
}

//Capabilities wrapper around github.com/docker/go-plugins-helpers/volume
func Capabilities() *volume.CapabilitiesResponse {
	log.Debug().Msgf("Entering Capabilities")
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "global",
		},
	}
}
