package unzip

import (
	"os"
	"testing"
)

// func UnZip(inputFn string, tmpDir string) (fileList []string, err error) {
func Test_Unzip(t *testing.T) {

	os.RemoveAll("./tmp")
	os.Mkdir("./tmp", 0700)

	fns, err := UnZip("./testdata/a.zip", "./tmp")

	if err != nil {
		t.Errorf("Test_UnZip")
	}
	if len(fns) != 2 {
		t.Errorf("Test_UnZip")
	}

	os.RemoveAll("./tmp")

}
