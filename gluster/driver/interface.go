package driver

import (
	"github.com/sapk/docker-volume-helpers/driver"

	"github.com/docker/go-plugins-helpers/volume"
)

type GlusterMountpoint struct {
	Path        string `json:"path"`
	Connections int    `json:"connections"`
}

func (d *GlusterMountpoint) GetPath() string {
	return d.Path
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

//List volumes handled by these driver
func (d *GlusterDriver) List() (*volume.ListResponse, error) {
	return driver.List(d)
}

//Get get info on the requested volume
func (d *GlusterDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	v, m, err := driver.Get(d, r.Name)
	if err != nil {
		return nil, err
	}
	return &volume.GetResponse{Volume: &volume.Volume{Name: r.Name, Status: v.GetStatus(), Mountpoint: m.GetPath()}}, nil
}

//Remove remove the requested volume
func (d *GlusterDriver) Remove(r *volume.RemoveRequest) error {
	return driver.Remove(d, r.Name)
}

//Unmount unmount the requested volume
func (d *GlusterDriver) Unmount(r *volume.UnmountRequest) error {
	return driver.Unmount(d, r.Name)
}

//Capabilities Send capabilities of the local driver
func (d *GlusterDriver) Capabilities() *volume.CapabilitiesResponse {
	return driver.Capabilities()
}
