package driver

import "testing"

func TestInit(t *testing.T) {
	d := Init("/tmp/test-root", "")
	if d == nil {
		t.Error("Expected to be not null, got ", d)
	}
	/*
		  if _, err := os.Stat(cfgFolder + "gluster-persistence.json"); err != nil {
				t.Error("Expected file to exist, got ", err)
			}
	*/
}
