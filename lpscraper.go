package main

import (
	"sync"

	"lpscraper/scraper"
	. "lpscraper/utils"
	"lpscraper/log"
)

var wg sync.WaitGroup
var locations = []string{
	"england/london",
	"england",
	"scotland",
	"wales",
	"great-britain",
	"the-united-kingdom",
	"scotland/edinburgh",
}

func main() {
	// This ensures that any panics to trigger OS exits lead to
	// OS exits after other deferred functions are run.
	defer HandleExit()
	defer log.File.Close()

	for _, location := range locations {
		wg.Add(1)
		go scrape(location)
	}

	wg.Wait()
}

// TODO handle panics - this is not good.
func scrape(location string) {
	scr := scraper.NewLpScraper(location)

	resp, err := scr.Load()
	if err != nil {
		log.Info.Printf("Could not open webpage %s: %v\n", scr.OID(), err)
		panic(Exit{Code: 1})
	}
	log.Info.Printf("Loaded webpage %s\n", scr.OID())
	defer resp.Body.Close()

	err = scr.Parse(resp)
	if err != nil {
		log.Info.Printf("Could not parse webpage %s: %v\n", scr.OID(), err)
		panic(Exit{Code: 1})
	}
	log.Info.Printf("Parsed webpage %s\n", scr.OID())

	scr.Download()
	wg.Done()
}

