package index

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"www.2c-why.com/nuvi/news-aggregator/naLib"
)

func Test_GetDirectory(t *testing.T) {
	gCfg := naLib.GlobalConfigType{
		RedisHost:                   "127.0.0.1",
		RedisPort:                   "6379",
		RedisAuth:                   "",
		RunFreq:                     0,
		ServiceName:                 "news:aggrigator",
		RedisPrefix:                 "",
		RedisKeySetOfFilesDownoaded: "downloaded-files",
		RedisKeyLoadedDocuments:     "loaded-documents",
		TmpDir:                      "./tmp",
		TmpPrefix:                   "na_",
		RedisKeyNewsXML:             "NEWS_XML",
	}
	naLib.ReadConfigFile("../cfg.json", &gCfg)
	gCfg.DebugFlags = make(map[string]bool) // turn off all debug flags for this test
	gCfg.LoadUrl = "http://localhost:19191"
	ex := `Some test Data`
	os.Mkdir("./testdata", 0700)
	ioutil.WriteFile("./testdata/test01.txt", []byte(ex), 0600)
	go func() {
		http.Handle("/", http.FileServer(http.Dir("./testdata")))
		log.Println("Server started: http://localhost:19191")
		log.Fatal(http.ListenAndServe(":19191", nil))
	}()

	data, err := GetDirectory(gCfg.LoadUrl + "/")

	// fmt.Printf("err=%s, [%s]\n", err, data)
	if err != nil {
		t.Errorf("Test_GetDirectory")
	}

	if string(data) != `<pre>
<a href="test01.txt">test01.txt</a>
</pre>
` {
		t.Errorf("Test_GetDirectory")
	}

	os.RemoveAll("./testdata")
}

// func ParseDirectory(data []byte) (fns []string, err error) {
func Test_ParseDirectory(t *testing.T) {
	data := []byte(`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN">
<html>
 <head>
  <title>Index of /5Rh5AMTrc4Pv/mainstream/posts</title>
 </head>
 <body>
<h1>Index of /5Rh5AMTrc4Pv/mainstream/posts</h1>
<table><tr><th><a href="?C=N;O=D">Name</a></th><th><a href="?C=M;O=A">Last modified</a></th><th><a href="?C=S;O=A">Size</a></th><th><a href="?C=D;O=A">Description</a></th></tr><tr><th colspan="4"><hr></th></tr>
<tr><td><a href="/5Rh5AMTrc4Pv/mainstream/">Parent Directory</a></td><td>&nbsp;</td><td align="right">  - </td><td>&nbsp;</td></tr>
<tr><td><a href="1471622300928.zip">1471622300928.zip</a></td><td align="right">19-Aug-2016 19:02  </td><td align="right">9.9M</td><td>&nbsp;</td></tr>
<tr><td><a href="1471622554118.zip">1471622554118.zip</a></td><td align="right">19-Aug-2016 19:05  </td><td align="right">9.9M</td><td>&nbsp;</td></tr>
<tr><th colspan="4"><hr></th></tr>
</table>
</body></html>`)

	fns, err := ParseDirectory(data)

	if err != nil {
		t.Errorf("Test_ParseDirectory")
	}
	if len(fns) != 2 {
		t.Errorf("Test_ParseDirectory")
	}

}
