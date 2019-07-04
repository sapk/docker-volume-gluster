package driver

import (
	"testing"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sapk/docker-volume-helpers/basic"
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
		{"192.168.1.1,test2:volume/subdir", true},
		{"192.168.1.1,test2:/volume/subdir", true},
		{"192.168.1.1,test2://volume/subdir", false},
		{"192.168.1.1,test2:/volume", true},
		{"مثال.إختبار:volume", true},
		{"例子.测试:volume", true},
		{"例子.測試:volume", true},
		{"παράδειγμα.δοκιμή:volume", true},
		{"उदाहरण.परीक्षा:volume", true},
		{"例え.テスト:volume", true},
		{"실례.테스트:volume", true},
		{"مثال.آزمایشی:volume", true},
		{"пример.испытание:volume", true},
	}

	for _, test := range tt {
		r := isValidURI(test.value)
		if test.result != r {
			t.Errorf("Expected URI '%s' to be '%v' , got '%v'", test.value, test.result, r)
		}
	}
}
func TestParseVolURI(t *testing.T) {
	tt := []struct {
		value  string
		result string
	}{
		{"test:volume", "--volfile-id='volume' -s 'test'"},
		{"test:/volume", "--volfile-id='volume' -s 'test'"},
		{"test:/volume/subdir", "--volfile-id='volume' --subdir-mount='/subdir' -s 'test'"},
		{"test:/volume/subdir/dir", "--volfile-id='volume' --subdir-mount='/subdir/dir' -s 'test'"},
		{"test,test2:volume", "--volfile-id='volume' -s 'test' -s 'test2'"},
		{"192.168.1.1:volume", "--volfile-id='volume' -s '192.168.1.1'"},
		{"192.168.1.1,10.8.0.1:volume", "--volfile-id='volume' -s '192.168.1.1' -s '10.8.0.1'"},
		{"192.168.1.1,test2:volume", "--volfile-id='volume' -s '192.168.1.1' -s 'test2'"},
	}

	for i, test := range tt {
		r := parseVolURI(test.value)
		if test.result != r {
			t.Errorf("Expected test %d to be '%v' , got '%v'", i, test.result, r)
		}
	}
}

func TestMountName(t *testing.T) {
	name, err := GetMountName(&basic.Driver{
		Config: &basic.DriverConfig{
			CustomOptions: map[string]interface{}{
				"mountUniqName": false,
			},
		},
	}, &volume.CreateRequest{
		Name: "test",
		Options: map[string]string{
			"voluri": "gluster-node:volname",
		},
	})

	if err != nil {
		t.Error("Expected to be null, got ", err)
	}

	if name != "test" {
		t.Error("Expected to be test, got ", name)
	}

	nameuniq, err := GetMountName(&basic.Driver{
		Config: &basic.DriverConfig{
			CustomOptions: map[string]interface{}{
				"mountUniqName": true,
			},
		},
	}, &volume.CreateRequest{
		Name: "test",
		Options: map[string]string{
			"voluri": "gluster-node:volname",
		},
	})

	if err != nil {
		t.Error("Expected to be null, got ", err)
	}

	if nameuniq != "gluster-node:volname" {
		t.Error("Expected to be gluster-node:volname, got ", name)
	}
}
