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
	"io/ioutil"
	"log"
	"time"

	"github.com/pschlump/Go-FTL/server/sizlib"

	"www.2c-why.com/nuvi/news-aggregator/index"
)

// "www.2c-why.com/nuvi/news-aggregator/unzip"

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
	RedisAuth                   string          `json:"RedisAuth"`                   //
	RunFreq                     int             `json:"RunFreq"`                     //
	ServiceName                 string          `json:"ServiceName"`                 //
	RedisPrefix                 string          `json:"RedisPrefix"`                 //
	RedisKeySetOfFilesDownoaded string          `json:"RedisKeySetOfFilesDownoaded"` //
	RedisKeyLoadedDocuments     string          `json:"RedisKeyLoadedDocuments"`     //
	RunMode                     string          `json:"RunMode"`                     //
	DebugFlags                  map[string]bool `json:"DebugFlags"`                  //
	LoadUrl                     string          `json:"LoadUrl"`                     //
}

var gCfg GlobalConfigType

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

	// iterate in a loop if RunFreq > 0, else just run once
	if gCfg.RunFreq > 0 {
		for n := 1; ; n++ {
			fmt.Printf("Running every %d seconds, iteration %d\n", gCfg.RunFreq, n)
			RunMainProcess()
			time.Sleep(time.Duration(gCfg.RunFreq) * time.Second)
		}
	} else {
		fmt.Printf("Running just onece\n")
		RunMainProcess()
	}

}

func RunMainProcess() {

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

	// xyzzy - connect to Redis

	// xyzzy - remove duplciates for download (if dbOnly1File, then only run 1 file) -- if Rerun - then search for that file
	if Rerun != nil && len(*Rerun) > 0 {
		if sizlib.InArray(*Rerun, fList) {
			fList = []string{*Rerun}
		} else {
			log.Printf("Unable to rerun %s - file is not available.", *Rerun)
			return
		}
	}
	fList = RemoveDuplicateDownloadFiles(fList)
	if IsDbOn("dbOnly1File", &gCfg) { // this is for testing - to only run 1 file
		if len(fList) > 1 {
			fmt.Printf("Debug flag %s is on, only run 1 file, list reduced from %s to %s\n", "dbOnly1File", fList, fList[0:1])
			fList = fList[0:1]
		}
	}

	// xyzzy - probably need to download into a tmp directory
	// download files form list -- may want to do this in parallel --
	DownloadZipFiles(fList)

	// xyzzy - create temporary directory for each file to extract into - one temporary for each file - (if db2 then leave directory after run)

	// xyzzy - extract each .zip file - get list of file names. (if db3 then leave .zip file, else if no error then discard)

	// xyzzy - for each xml in .zip file

	//		xyzzy - if it is not alredy loaded

	//		xyzzy - load into list in redis

}

// IsDbOn returns true if a specified debugt flag is enabled.
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

func RemoveDuplicateDownloadFiles(fList []string) (rv []string) {
	// xyzzy - use Redis set to see if fiel is already down.
	// xyzzy - possibly check file system for it.
	rv = fList
	return
}

func DownloadZipFiles(fList []string) {
}
