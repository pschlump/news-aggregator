package index

// data, err := index.GetDirectory(gCfg.LoadUrl)
func GetDirectory(URL string) (data []byte, err error) {
	return
}

// ParseDirectory takes a directory listing from a file server and extracts the list of file names.
// TODO: this could be significanly improved using a HTML parser - instead of ``greping'' the file names.
// that is a 2-4 day project to get it working, however that will significantly improve the reliability of this.
func ParseDirectory(data []byte) (fns []string, err error) {
	return
}
