package main

import (
	"flag"
	"log"
	"os"

	"github.com/chapsuk/crawler"
)

var (
	endpoint = flag.String("h", "https://github.com/chapsuk", "base endpoint")
	out      = flag.String("o", "./result/", "output path")
	workers  = flag.Int("w", 150, "workers count")
	resume   = flag.Bool("r", false, "resume upload")
	subdom   = flag.Bool("s", false, "include subdomains")
	gzip     = flag.Bool("g", true, "enable gzip")
)

func main() {
	flag.Parse()

	if *endpoint == "" || *out == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	err := createOutput(*out)
	if err != nil {
		log.Panic(err)
	}

	c, err := crawler.New(*endpoint, *out)
	if err != nil {
		log.Panic(err)
	}

	c.IncludeSubDomains = *subdom
	c.SaveWorkers = *workers
	c.UploadWorkers = *workers
	c.EnableGzip = *gzip
	c.Run(*resume)
}

func createOutput(path string) error {
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return os.MkdirAll(path, 0744)
	}
	return nil
}
