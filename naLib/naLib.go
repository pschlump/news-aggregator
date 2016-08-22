package naLib

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/radix.v2/redis"
)

// GlobalConfigType is for reading in the configuration for this program.
// Example:
// {
// 	"RedisHost":  "192.168.0.133",
// 	"RedisAuth":  "lLJSmkccYJiVEwskr1RM4MWIaBM",
// 	"RunFreq": 30,
// 	"ServiceName": "news:aggrigator",
// 	"RedisPrefi: "na:",
// 	"RedisKeySetOfFilesDownoaded": "downloaded-files",
// 	"RedisKeyLoadedDocuments": "loaded-documents",
// 	"RunMode": "test",
// 	"DebugFlags": [ "db1" ]
// }

type GlobalConfigType struct {
	RedisHost                   string          `json:"RedisHost"`                   //
	RedisPort                   string          `json:"RedisPort"`                   //
	RedisAuth                   string          `json:"RedisAuth"`                   //
	RunFreq                     int             `json:"RunFreq"`                     //
	ServiceName                 string          `json:"ServiceName"`                 //
	RedisPrefix                 string          `json:"RedisPrefix"`                 //
	RedisKeySetOfFilesDownoaded string          `json:"RedisKeySetOfFilesDownoaded"` //
	RedisKeyLoadedDocuments     string          `json:"RedisKeyLoadedDocuments"`     //
	RunMode                     string          `json:"RunMode"`                     //
	DebugFlags                  map[string]bool `json:"DebugFlags"`                  //
	LoadUrl                     string          `json:"LoadUrl"`                     //
	TmpDir                      string          `json:"TmpDir"`                      //	Where to create temporary directories
	TmpPrefix                   string          `json:"TmpPrefix"`                   // Prefix to create the temporary directories with
	RedisKeyNewsXML             string          `json:"RedisKeyNewsXML"`             //
}

// IsDbOn returns true if a specified debug flag is enabled.
func IsDbOn(flag string, gCfg *GlobalConfigType) (on bool) {
	x, ok := gCfg.DebugFlags[flag] // since the flags are static during the run there is no need to have a lock on the map.
	on = ok && x
	return
}

// ReadConfigFile reads in JSON configuraiton file for this program.  Errors are fatal.
func ReadConfigFile(fn string, gCfg *GlobalConfigType) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatalf("Fatal: Unable to read configuration file %s", fn)
	}

	err = json.Unmarshal(data, gCfg)
	if err != nil {
		log.Fatalf("Fatal: Syntax error in JSON configuration file %s", fn)
	}
}

// RemoveDuplicateDownloadFiles takes a list of .zip files to be downloaded and looks in the list in Redis
// to see if the file has already been marked as downloaded.  The returned list is just the files that do not
// appear in the Redis list of files that have already been processed.
func RemoveDuplicateDownloadFiles(client *redis.Client, fList []string, gCfg *GlobalConfigType) (rv []string) {
	// Use Redis set to see if file is already down.
	for _, fn := range fList {
		key := gCfg.RedisPrefix + gCfg.RedisKeySetOfFilesDownoaded
		// TODO: this has a race condition in it - if multiple processes are to be run then this test should be changed to set a key, and check the key in Redis.
		if !IsInRedisSet(client, fn, key) {
			rv = append(rv, fn)
			AddToRedisSet(client, fn, key)
		}
	}
	return
}

// DownloadZipFiles downloads each of the files in the fList into tmpDir
func DownloadZipFiles(fList []string, tmpDir string, gCfg *GlobalConfigType) (fullPathFn []string) {
	for _, fn := range fList {

		fpfn := tmpDir + "/" + fn
		fullPathFn = append(fullPathFn, fpfn)

		fp, err := sizlib.Fopen(fpfn, "w")
		if err != nil {
			log.Printf("Error: Unable to open file %s", fpfn)
		} else {
			defer fp.Close()

			URL := gCfg.LoadUrl + "/" + fn
			HTTPGetToFile(URL, fp, fpfn)
		}
	}
	return
}

// HTTPGetToFile will perform a http.Get on the specified url, then copying the data to the file fp/fn.
func HTTPGetToFile(URL string, fp *os.File, fn string) (status int) {
	res, err := http.Get(URL)
	if err != nil {
		log.Printf("Error: Unable to http.Get url %s", URL)
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		_, err := io.Copy(fp, res.Body)
		if err != nil {
			log.Printf("Error: Unable to copy data to %s", fn)
			return
		}
	} else {
		log.Printf("Error: Failed to get %s, got status of %d", URL, res.StatusCode)
	}
	status = res.StatusCode
	return
}

// RedisClient connects to Redis or returns an error.
func RedisClient(RedisHost, RedisPort, RedisAuth string) (client *redis.Client, err error) {
	client, err = redis.Dial("tcp", RedisHost+":"+RedisPort)
	if err != nil {
		return
	}
	if RedisAuth != "" {
		err = client.Cmd("AUTH", RedisAuth).Err
		if err != nil {
			return
		}
	}
	return
}

// IsInRedisSet returns true if 'item' is in the Redis set 'key'.
func IsInRedisSet(client *redis.Client, item, key string) bool {
	// - use sets, SISMEMBER - to find if in set
	n, err := client.Cmd("SISMEMBER", key, item).Int()
	if err != nil {
		log.Printf("Error: Redis SISMEMBER, %s, %s returned error %s\n", key, item, err)
		return false
	}
	if n == 1 {
		return true
	}
	return false
}

// AddToRedisSet will add the specified 'item' to the redis set 'key'.
func AddToRedisSet(client *redis.Client, item, key string) {
	err := client.Cmd("SADD", key, item).Err
	if err != nil {
		log.Printf("Error: Redis SADD, %s, %s returned error %s\n", key, item, err)
	}
}

// RedisLoadFile will take the contents of the file 'fn' and LPUSH it onto the Redis list specified by listKey
func RedisLoadFile(client *redis.Client, listKey string, fn string, gCfg *GlobalConfigType) {

	data, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Printf("Error: Failed to read %s, error=%s", fn, err)
		return
	}

	if IsDbOn("dbSkipPushOfContent", gCfg) { // this is for testing - leave temporary directory in place
		fmt.Printf("Skipping Redis: LPUSH %s len(data=%d, fn=%s)\n", listKey, len(data), fn)
		return
	}

	// This is assuming that you want to LPUSH and RPOP for processing.  So this adds to the "left" side of the list.
	err = client.Cmd("LPUSH", listKey, string(data)).Err
	if err != nil {
		log.Printf("Error: Redis LPUSH, %s, %s returned error %s\n", listKey, fn, err)
	}
}
