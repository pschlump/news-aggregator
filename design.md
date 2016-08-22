1. Specify link on CLI
	Specify cfg file on CLI

2. Download list of .zip files (only download any new files in list)
3. Unzip the new files
4. For each item in the unziped list process the .xml files - report any extraneous files
5. Keep a list of the HASH of each file that is processed - skip any already processed files
7. Make a list and create a searchable data set of the list.

Project Breakdown

main
	- read in a JSON config file with Redis connect info
	- connect to Redis
	- command cli -to- process files
	- call each section

getFileList
	- Given URL get list of files from the directory list

filterList
	- Look at a list and comapre to redis list - eliminate duplicates, return any new

getZip 
	- Given a list of files download each one

extractZip
	- Unzip a .zip file into a directory

findFiles 
	- Find all the .xml files in a direcotry

fileHash
	- Hash the file and get a SHA of the file

- See if file is duplicate

readXmlFile 
	- read in an xml file
	- Load to Redis
	


Assumptions:
	1. The .xml files are stored under the md5sum of the contents of the file.  -assumption-1-

1000 files at 10mb per zip - converts to -
	50bm of xml per file,
	1000*50mb = 50,000mb = 50T per set.


na:downloaded-files		-- list of files that we already have
na:loaded-documents		-- list of documents that we already have loaded

- use sets, SISMEMBER - to find if in set
	SADD - to add to set


