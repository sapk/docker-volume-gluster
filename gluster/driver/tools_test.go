package driver

import (
	"testing"
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
		{"test:volume", "--volfile-id='volume' -s 'test'"},
		{"test,test2:volume", "--volfile-id='volume' -s 'test' -s 'test2'"},
		{"192.168.1.1:volume", "--volfile-id='volume' -s '192.168.1.1'"},
		{"192.168.1.1,10.8.0.1:volume", "--volfile-id='volume' -s '192.168.1.1' -s '10.8.0.1'"},
		{"192.168.1.1,test2:volume", "--volfile-id='volume' -s '192.168.1.1' -s 'test2'"},
	}

	for _, test := range tt {
		r := parseVolURI(test.value)
		if test.result != r {
			t.Errorf("Expected to be '%v' , got '%v'", test.result, r)
		}
	}
}
