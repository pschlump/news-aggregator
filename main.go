package main

//
// News agrigator
// Author: Philip Schlump
// github.com:   https://github.com/pschlump/news-aggregator.git
//

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pschlump/Go-FTL/server/sizlib"
	"github.com/pschlump/radix.v2/redis"

	"www.2c-why.com/nuvi/news-aggregator/index"
	"www.2c-why.com/nuvi/news-aggregator/unzip"
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

var gCfg = GlobalConfigType{
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

var Rerun = flag.String("rerun", "", "Rerun of a specific .zip file")                  //
var URL = flag.String("URL", "", "Load from URL - overrides default in cfg.json file") //
var Cfg = flag.String("cfg", "cfg.json", "Configuraiton and Redis connection info")    //
func init() {
	flag.StringVar(Rerun, "r", "", "Rerun of a specific .zip file")                    //
	flag.StringVar(URL, "u", "", "Load from URL - overrides default in cfg.json file") //
	flag.StringVar(Cfg, "c", "cfg.json", "Configuraiton and Redis connection info")    //
}

func main() {

	flag.Parse()

	// read in config file
	ReadConfigFile(*Cfg, &gCfg)

	fmt.Printf("Servcie: [%s] Started\n", gCfg.ServiceName)

	// override configuration items from command line ( -u flag )
	if URL != nil && len(*URL) > 0 {
		gCfg.LoadUrl = *URL
	}

	// connect to Redis
	client, err := RedisClient(gCfg.RedisHost, gCfg.RedisPort, gCfg.RedisAuth)
	if err != nil {
		log.Printf("Unable to connect to Redis, error=%s", err)
		return
	}

	os.Mkdir(gCfg.TmpDir, 0700)

	// iterate in a loop if RunFreq > 0, else just run once
	if gCfg.RunFreq > 0 {
		for n := 1; ; n++ {
			fmt.Printf("Running every %d seconds, iteration %d\n", gCfg.RunFreq, n)
			RunMainProcess(client)
			time.Sleep(time.Duration(gCfg.RunFreq) * time.Second)
		}
	} else {
		fmt.Printf("Running just onece\n")
		RunMainProcess(client)
	}

}

func RunMainProcess(client *redis.Client) {

	// get list of files -- directory listing via http.Get()
	data, err := index.GetDirectory(gCfg.LoadUrl)
	if err != nil {
		log.Printf("Unable to get directory from %s, error=%s", gCfg.LoadUrl, err)
		return
	}

	// parse to list of file names
	fList, err := index.ParseDirectory(data)
	if err != nil {
		log.Printf("Unable to parse directory from %s, error=%s", gCfg.LoadUrl, err)
		return
	}

	// remove duplciates for download (if dbOnly1File, then only run 1 file) -- if Rerun - then search for that file
	if Rerun != nil && len(*Rerun) > 0 {
		if sizlib.InArray(*Rerun, fList) {
			fList = []string{*Rerun}
		} else {
			log.Printf("Unable to rerun %s - file is not available.", *Rerun)
			return
		}
	}
	fList = RemoveDuplicateDownloadFiles(client, fList)
	if IsDbOn("dbOnly1File", &gCfg) { // this is for testing - to only run 1 file
		if len(fList) > 1 {
			fmt.Printf("Debug flag %s is on, only run 1 file, list reduced from %s to %s\n", "dbOnly1File", fList, fList[0:1])
			fList = fList[0:1]
		}
	}
	if len(fList) == 0 {
		fmt.Printf("No new files to process\n")
		return
	}
	fmt.Printf("Processing %s\n", fList)

	// probably need to download into a tmp directory -- Create the tmp-dir
	name, err := ioutil.TempDir(gCfg.TmpDir, gCfg.TmpPrefix)
	if err != nil {
		log.Printf("Unable to create temporary directory, error=%s", err)
		return
	}
	fmt.Printf("name=%s\n", name)

	// download files form list -- may want to do this in parallel --
	fpfnList := DownloadZipFiles(fList, name)

	for ii, zip := range fpfnList {
		// skip, just use "name" - create temporary directory for each file to extract into - one temporary for each file - (if db2 then leave directory after run)
		zipname, err := ioutil.TempDir(name, fList[ii]) // don't much like this.
		if err != nil {
			// xyzzy log
		} else {

			// extract each .zip file - get list of file names. (if dbLeaveTmpDir then leave .zip file, else if no error then discard)
			zipList, err := unzip.UnZip(zip, zipname)
			if err != nil {
				// xyzzy log
			}

			if IsDbOn("dbPrintListOfZipFiles", &gCfg) { // this is for testing - leave temporary directory in place
				fmt.Printf("for %s in %s list of .zip files = %s\n", zip, zipname, zipList)
			}

			// for each xml in .zip file -- use zipList
			for _, xmlfn := range zipList {
				//		if it is not already loaded
				key := gCfg.RedisPrefix + gCfg.RedisKeyLoadedDocuments
				// TODO: this has a race condition in it - if multiple processes are to be run then this test should be changed to set a key, and check the key in Redis.
				if !IsInRedisSet(client, xmlfn, key) {
					//		load into list in Redis - gCfg.RedisKeyNewsXML     : "NEWS_XML",
					AddToRedisSet(client, xmlfn, key)
					RedisLoadFile(client, gCfg.RedisKeyNewsXML, zipname+"/"+xmlfn)
				}
			}

			// cleanup temporary files
			if !IsDbOn("dbLeaveTmpDir", &gCfg) { // this is for testing - leave temporary directory in place
				os.RemoveAll(zipname)
			}
		}
	}

	// cleanup - remove temporary directories
	if !IsDbOn("dbLeaveTmpDir", &gCfg) { // this is for testing - leave temporary directory in place
		os.RemoveAll(name)
	}
}

// IsDbOn returns true if a specified debug flag is enabled.
func IsDbOn(flag string, gCfg *GlobalConfigType) (on bool) {
	// xyzzy lock for debug flags map
	x, ok := gCfg.DebugFlags[flag]
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

func RemoveDuplicateDownloadFiles(client *redis.Client, fList []string) (rv []string) {
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

// download files form list -- may want to do this in parallel --
func DownloadZipFiles(fList []string, tmpDir string) (fullPathFn []string) {
	for _, fn := range fList {

		fpfn := tmpDir + "/" + fn
		fullPathFn = append(fullPathFn, fpfn)

		fp, err := sizlib.Fopen(fpfn, "w")
		if err != nil {
			// xyzzy
		}
		defer fp.Close()

		URL := gCfg.LoadUrl + "/" + fn
		HTTPGetToFile(URL, fp)
	}
	return
}

func HTTPGetToFile(URL string, fp *os.File) (status int) {
	res, err := http.Get(URL)
	if err != nil {
		// xyzzy - log
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		_, err := io.Copy(fp, res.Body)
		if err != nil {
			// xyzzy - log
			return
		}
	} else {
		// xyzzy - log - status - why getting non-200 status codes
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

func IsInRedisSet(client *redis.Client, fn, key string) bool {
	// - use sets, SISMEMBER - to find if in set
	n, err := client.Cmd("SISMEMBER", key, fn).Int()
	if err != nil {
		log.Printf("Error: Redis SISMEMBER, %s, %s returned error %s\n", key, fn, err)
		return false
	}
	if n == 1 {
		return true
	}
	return false
}

func AddToRedisSet(client *redis.Client, fn, key string) {
	err := client.Cmd("SADD", key, fn).Err
	if err != nil {
		log.Printf("Error: Redis SADD, %s, %s returned error %s\n", key, fn, err)
	}
}

func RedisLoadFile(client *redis.Client, listKey string, fn string) {

	data, err := ioutil.ReadFile(fn)
	if err != nil {
		// xyzzy - log
	}

	if IsDbOn("dbSkipPushOfContent", &gCfg) { // this is for testing - leave temporary directory in place
		fmt.Printf("Skipping Redis: LPUSH %s len(data=%d, fn=%s)\n", listKey, len(data), fn)
		return
	}

	// This is assuming that you want to LPUSH and RPOP for processing.  So this adds to the "left" side of the list.
	err = client.Cmd("LPUSH", listKey, string(data)).Err
	if err != nil {
		log.Printf("Error: Redis LPUSH, %s, %s returned error %s\n", listKey, fn, err)
	}
}
