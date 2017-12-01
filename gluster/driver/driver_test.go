package driver

import (
	"testing"

	"github.com/docker/go-plugins-helpers/volume"
)

func TestInit(t *testing.T) {
	d := Init("/tmp/test-root", "", false)
	if d == nil {
		t.Error("Expected to be not null, got ", d)
	}
	/*
		  if _, err := os.Stat(cfgFolder + "gluster-persistence.json"); err != nil {
				t.Error("Expected file to exist, got ", err)
			}
	*/
}

func TestMountName(t *testing.T) {
	name := getMountName(&GlusterDriver{
		mountUniqName: false,
	}, volume.CreateRequest{
		Name: "test",
		Options: map[string]string{
			"voluri": "gluster-node:volname",
		},
	})

	if name != "test" {
		t.Error("Expected to be test, got ", name)
	}

	nameuniq := getMountName(&GlusterDriver{
		mountUniqName: true,
	}, volume.CreateRequest{
		Name: "test",
		Options: map[string]string{
			"voluri": "gluster-node:volname",
		},
	})

	if nameuniq != "gluster-node:volname" {
		t.Error("Expected to be gluster-node:volname, got ", name)
	}
}
