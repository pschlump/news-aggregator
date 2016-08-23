package naLib

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
)

func Test_IsDbOn(t *testing.T) {
	gCfg := GlobalConfigType{
		DebugFlags: map[string]bool{
			"x1": true,
			"x2": false,
		},
	}
	// func IsDbOn(flag string, gCfg *GlobalConfigType) (on bool) {
	if !IsDbOn("x1", &gCfg) {
		t.Errorf("IsDbOn error\n")
	}
	if IsDbOn("x2", &gCfg) {
		t.Errorf("IsDbOn error\n")
	}
	if IsDbOn("x3", &gCfg) {
		t.Errorf("IsDbOn error\n")
	}
}

func Test_ReadConfigFile(t *testing.T) {
	os.Mkdir("./testdata", 0700)
	ioutil.WriteFile("./testdata/cfg-test.json", []byte(`{
	"RedisHost":  "192.168.0.3",
	"RedisAuth":  "*password*"
}`), 0600)
	gCfg := GlobalConfigType{
		DebugFlags: map[string]bool{
			"x1": true,
			"x2": false,
		},
	}

	ReadConfigFile("./testdata/cfg-test.json", &gCfg)

	if gCfg.RedisHost != "192.168.0.3" {
		t.Errorf("ReadConfigFile error\n")
	}
	if !IsDbOn("x1", &gCfg) {
		t.Errorf("ReadConfigFile error\n")
	}

	os.RemoveAll("./testdata")
}

// Tests:
// 	func RemoveDuplicateDownloadFiles(client *redis.Client, fList []string, gCfg *GlobalConfigType) (rv []string) {
// 	func IsInRedisSet(client *redis.Client, item, key string) bool {
// 	func AddToRedisSet(client *redis.Client, item, key string) {
//	func RedisClient(RedisHost, RedisPort, RedisAuth string) (client *redis.Client, err error) {
//
// test depends on connecting to Redis and the ../cfg.json file
func Test_RemoveDuplicateDownloadFiles(t *testing.T) {
	gCfg := GlobalConfigType{
		RedisHost:                   "127.0.0.1",
		RedisPort:                   "6379",
		RedisAuth:                   "",
		RunFreq:                     0,
		ServiceName:                 "news:aggrigator",
		RedisPrefix:                 "",
		RedisKeySetOfFilesDownoaded: "test-downloaded-files",
		RedisKeyLoadedDocuments:     "test-loaded-documents",
		TmpDir:                      "./tmp",
		TmpPrefix:                   "na_",
		RedisKeyNewsXML:             "NEWS_XML",
	}
	ReadConfigFile("../cfg.json", &gCfg)
	gCfg.RedisPrefix = ""
	gCfg.RedisKeySetOfFilesDownoaded = "test-downloaded-files"

	// connect to Redis
	client, err := RedisClient(gCfg.RedisHost, gCfg.RedisPort, gCfg.RedisAuth)
	if err != nil {
		t.Errorf("RedisClient error- failed to connect- %s\n", err)
		return
	}

	// Empty out the set
	for {
		s, err := client.Cmd("SPOP", "test-downloaded-files").Str()
		if err != nil {
			break
		}
		if s == "" {
			break
		}
	}

	fList := []string{"a.zip", "b.zip", "c.zip"}
	rv := RemoveDuplicateDownloadFiles(client, fList, &gCfg)
	if len(rv) != 3 {
		t.Errorf("RemoveDuplicateDownloadFiles error - expected 3, got %d\n", len(rv))
	}
	rv = RemoveDuplicateDownloadFiles(client, fList, &gCfg)
	if len(rv) != 0 {
		t.Errorf("RemoveDuplicateDownloadFiles error - expected 0, got %d\n", len(rv))
	}
	fList = []string{"a.zip", "b.zip", "c.zip", "d.zip"}
	rv = RemoveDuplicateDownloadFiles(client, fList, &gCfg)
	if len(rv) != 1 {
		t.Errorf("RemoveDuplicateDownloadFiles error - expected 1, got %d\n", len(rv))
	}

}

// func RedisLoadFile(client *redis.Client, listKey string, fn string, gCfg *GlobalConfigType) {
func Test_RedisLoadFile(t *testing.T) {
	gCfg := GlobalConfigType{
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
	ReadConfigFile("../cfg.json", &gCfg)
	gCfg.DebugFlags = make(map[string]bool) // turn off all debug flags for this test
	gCfg.RedisPrefix = ""
	gCfg.RedisKeyNewsXML = "test-NEWS_XML"

	// connect to Redis
	client, err := RedisClient(gCfg.RedisHost, gCfg.RedisPort, gCfg.RedisAuth)
	if err != nil {
		t.Errorf("RedisClient error- failed to connect- %s\n", err)
		return
	}

	ex := `Some test Data`

	os.Mkdir("./testdata", 0700)
	ioutil.WriteFile("./testdata/test01.txt", []byte(ex), 0600)

	key := "Test_RedisLoadFile:1"
	client.Cmd("DEL", key)

	RedisLoadFile(client, key, "./testdata/test01.txt", &gCfg)

	s, err := client.Cmd("RPOP", key).Str()
	if err != nil {
		t.Errorf("RedisLoadFile error- get returned error- %s\n", err)
	}
	if s != ex {
		t.Errorf("RedisLoadFile error- expected [%s] got [%s]\n", ex, s)
	}

	client.Cmd("DEL", key)
	os.RemoveAll("./testdata")
}

// Tests:
// 		func DownloadZipFiles(fList []string, tmpDir string, gCfg *GlobalConfigType) (fullPathFn []string) {
// 		func HTTPGetToFile(URL string, fp *os.File, fn string) (status int) {
func Test_DownloadZipFiles(t *testing.T) {
	gCfg := GlobalConfigType{
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
	ReadConfigFile("../cfg.json", &gCfg)
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

	os.Mkdir("./tmp", 0700)

	fList := []string{"test01.txt"}

	fp := DownloadZipFiles(fList, "./tmp", &gCfg)
	if len(fp) != 1 {
		t.Errorf("Test_DownloadZipFiles")
	}

	os.RemoveAll("./testdata")
}
