package index

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

var ErrUnableToGetIndex = errors.New("Unable to get directory index")

// data, err := index.GetDirectory(gCfg.LoadUrl)
func GetDirectory(URL string) (data []byte, err error) {
	var status int
	status, data = HTTPGet(URL)
	if status != http.StatusOK {
		err = ErrUnableToGetIndex
	}
	return
}

var matchLine *regexp.Regexp

func init() {
	matchLine = regexp.MustCompile(`<tr><td><a href="([0-9][0-9]*.zip)">`)
}

// ParseDirectory takes a directory listing from a file server and extracts the list of file names.
// TODO: this could be significanly improved using a HTML parser - instead of ``greping'' the file names.
// that is a 2-4 day project to get it working, however that will significantly improve the reliability of this.
func ParseDirectory(data []byte) (fns []string, err error) {
	// Example Line: <tr><td><a href="1471622300928.zip">1471622300928.zip</a></td><td align="right">19-Aug-2016 19:02  </td><td align="right">9.9M</td><td>&nbsp;</td></tr>
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		m := matchLine.FindAllStringSubmatch(line, -1)
		if len(m) > 0 {
			if len(m[0]) > 1 {
				fns = append(fns, m[0][1])
			}
		}
	}
	return
}

// TODO: may be better to have a streaming return on this - but ... this is simple for testing.
func HTTPGet(URL string) (status int, rv []byte) {
	res, err := http.Get(URL)
	if err != nil {
		return 500, []byte("")
	} else {
		defer res.Body.Close()
		var err error
		rv, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return 500, []byte("")
		}
		status = res.StatusCode
		return
	}
}
