package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/chapsuk/crawler"
)

var (
	endpoint = flag.String("h", "https://github.com/chapsuk", "base endpoint")
	out      = flag.String("o", "./result/", "output path")
	workers  = flag.Int("w", 150, "workers count")
	resume   = flag.Bool("r", false, "resume upload")
	subdom   = flag.Bool("s", false, "include subdomains")
	gzip     = flag.Bool("g", true, "enable gzip")
	db       = flag.String("d", "postgres://postgres:postgres@postgres/crawler?sslmode=disable", "db connections string")
)

func main() {
	flag.Parse()
	start := time.Now()

	if *endpoint == "" || *out == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	err := createOutput(*out)
	if err != nil {
		log.Panic(err)
	}

	m, err := url.Parse(*endpoint)
	if err != nil {
		log.Panicf("parse %s error: %s", *endpoint, err)
	}

	var state *crawler.State
	if *db == "" {
		state = crawler.NewState(nil)
	} else {
		strg, err := crawler.NewPGStorage(*db, m.Host)
		if err != nil {
			log.Printf("create storage error: %s", err)
			strg = nil
		}

		if strg == nil {
			state = crawler.NewState(nil)
		} else if *resume {
			state, err = strg.Load()
			if err != nil {
				log.Panicf("load state error: %s", err)
			}
		} else {
			state, err = strg.Clear()
			if err != nil {
				log.Panicf("clear state error: %s", err)
			}
		}
	}

	c, err := crawler.New(*endpoint, *out, state)
	if err != nil {
		log.Panic(err)
	}
	defer c.Close()

	c.IncludeSubDomains = *subdom
	c.SaveWorkers = *workers
	c.UploadWorkers = *workers
	c.EnableGzip = *gzip
	c.Run()

	log.Printf("Completed! Time: %s", time.Now().Sub(start).String())
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
