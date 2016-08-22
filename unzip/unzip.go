package unzip

import (
	"archive/zip"
	"errors"
	"io"

	"github.com/pschlump/Go-FTL/server/sizlib"
)

var ErrEmptyArchive = errors.New("Empty Archive")

// UnZip takes an input file name and un-zips it into the specified directory.  A list of files or an error is returned.
// If this is an empty arcive, then an error will be returnd.
func UnZip(inputFn string, tmpDir string) (fileList []string, err error) {

	// Open a zip archive for reading.
	r, err := zip.OpenReader(inputFn)
	if err != nil {
		return
	}
	defer r.Close()

	// Iterate through the files in the archive, and write each file out to the temporary directory.
	for _, f := range r.File {
		fileList = append(fileList, f.Name)
		// fmt.Printf("Contents of %s:\n", f.Name)
		fnContents := tmpDir + "/" + f.Name
		rc, err1 := f.Open()
		if err1 != nil {
			err = err1
			return
		}
		// Extra block to grantee that "defer"-red close will happen every loop, not at exit of function.
		{
			defer rc.Close()
			fo, err2 := sizlib.Fopen(fnContents, "w")
			if err2 != nil {
				err = err2
				return
			}
			defer fo.Close()
			_, err = io.Copy(fo, rc)
			if err != nil {
				return
			}
		}
	}

	if len(fileList) == 0 {
		err = ErrEmptyArchive
	}

	return
}
