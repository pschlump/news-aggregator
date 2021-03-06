package main

//
// News aggregate
// Author: Philip Schlump
// github.com:   https://github.com/pschlump/news-aggregator.git
//
// TODO:
//	1. 	A 2nd program that cleans up the old data in RedisKeySetOfFilesDownloaded.   This would be simple
//	  	since the file names are time stamps.   A cron job could do this.
//

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/pschlump/news-aggregator/index"
	"github.com/pschlump/news-aggregator/naLib"
	"github.com/pschlump/news-aggregator/unzip"
	"github.com/pschlump/radix.v2/redis"
)

var gCfg = naLib.GlobalConfigType{
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
	naLib.ReadConfigFile(*Cfg, &gCfg)

	if naLib.IsDbOn("dbVerbose", &gCfg) { // this is for testing - leave temporary directory in place
		fmt.Printf("Servcie: [%s] Started\n", gCfg.ServiceName)
	}

	// override configuration items from command line ( -u flag )
	if URL != nil && len(*URL) > 0 {
		gCfg.LoadUrl = *URL
	}

	// connect to Redis
	client, err := naLib.RedisClient(gCfg.RedisHost, gCfg.RedisPort, gCfg.RedisAuth)
	if err != nil {
		log.Printf("Unable to connect to Redis, error=%s", err)
		return
	}

	os.Mkdir(gCfg.TmpDir, 0700)

	// iterate in a loop if RunFreq > 0, else just run once
	if gCfg.RunFreq > 0 {
		for n := 1; ; n++ {
			if naLib.IsDbOn("dbVerbose", &gCfg) { // this is for testing - leave temporary directory in place
				fmt.Printf("Running every %d seconds, iteration %d\n", gCfg.RunFreq, n)
			}
			RunMainProcess(client)
			time.Sleep(time.Duration(gCfg.RunFreq) * time.Second)
		}
	} else {
		if naLib.IsDbOn("dbVerbose", &gCfg) { // this is for testing - leave temporary directory in place
			fmt.Printf("Running just once\n")
		}
		RunMainProcess(client)
	}

}

// RunMainProcess splits the main() into  2 parts to make it easy to process gCfg.RunFreq flag.
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

	// remove duplicates for download (if dbOnly1File, then only run 1 file) -- if Rerun - then search for that file
	if Rerun != nil && len(*Rerun) > 0 {
		if naLib.InArray(*Rerun, fList) {
			fList = []string{*Rerun}
		} else {
			log.Printf("Unable to rerun %s - file is not available.", *Rerun)
			return
		}
	}
	fList = naLib.RemoveDuplicateDownloadFiles(client, fList, &gCfg)
	if naLib.IsDbOn("dbOnly1File", &gCfg) { // this is for testing - to only run 1 file
		if len(fList) > 1 {
			fmt.Printf("Debug flag %s is on, only run 1 file, list reduced from %s to %s\n", "dbOnly1File", fList, fList[0:1])
			fList = fList[0:1]
		}
	}
	if len(fList) == 0 {
		fmt.Printf("No new files to process\n")
		return
	}
	if naLib.IsDbOn("dbVerbose", &gCfg) { // this is for testing - leave temporary directory in place
		fmt.Printf("Processing %s\n", fList)
	}

	// probably need to download into a tmp directory -- Create the tmp-dir
	name, err := ioutil.TempDir(gCfg.TmpDir, gCfg.TmpPrefix)
	if err != nil {
		log.Printf("Unable to create temporary directory, error=%s", err)
		return
	}
	if naLib.IsDbOn("dbVerbose", &gCfg) { // this is for testing - leave temporary directory in place
		fmt.Printf("Name=%s\n", name)
	}

	// download files form list -- may want to do this in parallel --
	fpfnList := naLib.DownloadZipFiles(fList, name, &gCfg)

	for ii, zip := range fpfnList {
		// skip, just use "name" - create temporary directory for each file to extract into - one temporary for each file - (if db2 then leave directory after run)
		zipname, err := ioutil.TempDir(name, fList[ii]) // don't much like this.
		if err != nil {
			log.Printf("Error: Unable to create temporary directory in %s", name)
		} else {

			// extract each .zip file - get list of file names. (if dbLeaveTmpDir then leave .zip file, else if no error then discard)
			zipList, err := unzip.UnZip(zip, zipname)
			if err != nil {
				log.Printf("Error: Unable to unzip %s", zip)
			} else {

				if naLib.IsDbOn("dbPrintListOfZipFiles", &gCfg) { // this is for testing - leave temporary directory in place
					fmt.Printf("for %s in %s list of .zip files = %s\n", zip, zipname, zipList)
				}

				// for each xml in .zip file -- use zipList
				for _, xmlfn := range zipList {
					//		if it is not already loaded
					key := gCfg.RedisPrefix + ":" + xmlfn
					if naLib.SetIfNotExists(client, xmlfn, key) {
						naLib.RedisLoadFile(client, gCfg.RedisKeyNewsXML, zipname+"/"+xmlfn, &gCfg)
					}
				}

				// cleanup temporary files
				if !naLib.IsDbOn("dbLeaveTmpDir", &gCfg) { // this is for testing - leave temporary directory in place
					os.RemoveAll(zipname)
				}

			}
		}
	}

	// cleanup - remove temporary directories
	if !naLib.IsDbOn("dbLeaveTmpDir", &gCfg) { // this is for testing - leave temporary directory in place
		os.RemoveAll(name)
	}
}
