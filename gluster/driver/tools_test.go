package driver

import (
	"strings"
	"testing"

	"github.com/docker/go-plugins-helpers/volume"
)

func TestIsValidURI(t *testing.T) {
	tt := []struct {
		value  string
		result bool
	}{
		{"test", false},
		{"test:volume", true},
		{"test;volume", false},
		{"test,volume", false},
		{"test,test2:volume", true},
		{"192.168.1.1:volume", true},
		{"192.168.1.:volume", false},
		{"192.168.1.1,10.8.0.1:volume", true},
		{"192.168.1.1,test2:volume", true},
	}

	for _, test := range tt {
		r := isValidURI(test.value)
		if test.result != r {
			t.Errorf("Expected to be '%v' , got '%v'", test.result, r)
		}
	}
}
func TestParseVolURI(t *testing.T) {
	tt := []struct {
		value  string
		result string
	}{
		{"test:volume", "--volfile-id=volume --volfile-server=test"},
		{"test,test2:volume", "--volfile-id=volume --volfile-server=test --volfile-server=test2"},
		{"192.168.1.1:volume", "--volfile-id=volume --volfile-server=192.168.1.1"},
		{"192.168.1.1,10.8.0.1:volume", "--volfile-id=volume --volfile-server=192.168.1.1 --volfile-server=10.8.0.1"},
		{"192.168.1.1,test2:volume", "--volfile-id=volume --volfile-server=192.168.1.1 --volfile-server=test2"},
	}

	for _, test := range tt {
		r := parseVolURI(test.value)
		if test.result != strings.Join(r, " ") {
			t.Errorf("Expected to be '%v' , got '%v'", test.result, r)
		}
	}
}

func TestMountName(t *testing.T) {
	name := getMountName(&GlusterDriver{
		mountUniqName: false,
	}, &volume.CreateRequest{
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
	}, &volume.CreateRequest{
		Name: "test",
		Options: map[string]string{
			"voluri": "gluster-node:volname",
		},
	})

	if nameuniq != "gluster-node:volname" {
		t.Error("Expected to be gluster-node:volname, got ", name)
	}
}
