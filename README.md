news-aggregator
===============

Process zip files with news articles and aggregate them into a list in Redis.

Some configuration is required.  The program has to be able to connect to Redis.
This is setup in the ./cfg.json file.

```JavaScript
{
	"RedisHost":  "192.168.0.133",
	"RedisPort":  "6379",
	"RedisAuth":  "lLJSmkccYJiVEwskr1RM4MWIaBM",
	"RunFreq": 0,
	"LoadUrl": "http://feed.omgili.com/5Rh5AMTrc4Pv/mainstream/posts/",
}
```

`RedisHost` is the IP address of the Redis server.  It defaults to 127.0.0.1.

`RedisPort` is the port the Redis server listens on.  It defaults to 6379, the default port for Redis.  It is a string.

`RedisAuth` is the authorization token for Redis.  It is optional and defaults to "".  If Redis is not configure to require authorization then it should be left as "".

`FunFreq` is a value in seconds.  If it is not 0 then the program will loop forever getting the list of .zip files to download and downloading them.

`LoadUrl` is the default location to get the list of .zip files from.  It can also be specified as -u on the command line.

To Install / Run
----------------

```
	$ git clone https://github.com/pschlump/news-aggregator.git
	$ cd news-aggregator
	$ go get
	$ go build
	$ vi cfg.json			# Adjust values as noted above
	$ ./news-aggregator
```

