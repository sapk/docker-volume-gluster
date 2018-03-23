package driver

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	f, err := ioutil.TempDir("", "testing")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(f) // clean up

	CfgFolder = f

	d := Init("/tmp/test-root", false)
	if d == nil {
		t.Error("Expected to be not null, got ", d)
	}
	log.Println(d.Config)

	d.SaveConfig()
	//Second reload should reload parsistence file
	d2 := Init("/tmp/test-root", false)
	if d2 == nil {
		t.Error("Expected to be not null, got ", d2)
	}
	log.Println(d2.Config)

	if _, err := os.Stat(CfgFolder + "/persistence.json"); err != nil {
		t.Error("Expected file to exist, got ", err)
	}

}
