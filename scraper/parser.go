package scraper

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"lpscraper/log"

	"github.com/PuerkitoBio/goquery"
)

// TODO Add parsing of child pages
// TODO Add handling retries
func (scr *lpScraper) Parse(resp *http.Response) error {
	log.Debug.Printf("Parse response for %s", scr.url.Path)
	logResponse(resp)

	if resp.StatusCode != 200 {
		log.Debug.Printf("Failed to load %s", scr.url.Path)
		return errors.New("Bad response code")
	}

	reader, err := newBodyReader(resp)
	if err != nil {
		log.Debug.Printf("Cannot read HTTP response for %s: %v", scr.url.Path, err)
		return err
	}
	defer reader.Close()

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Debug.Printf("Corrupted HTTP response for %s: %v", scr.url.Path, err)
		return err
	}

	scr.getImageData(doc)
	return nil
}

type image struct {
	URL   string `json:"medium"`
	Quote string `json:"strapline"`
}

func (img *Image) UnmarshalJSON(data []byte) error {
	i := &image{}
	err := json.Unmarshal(data, i)
	if err != nil {
		return err
	}

	URL, err := url.Parse(i.URL)
	if err != nil {
		return err
	}

	img.URL = URL
	img.URL.RawQuery = ""
	return nil
}

// Read the URLs for the desired images into scr.
func (scr *lpScraper) getImageData(webpage *goquery.Document) {
	log.Debug.Printf("Getting image data for %s\n", scr.url.Path)

	webpage.Find(".slideshow.js-slideshow").Each(func(i int, s *goquery.Selection) {
		// There should only be one of these, so skip otherwise.
		if i > 0 {
			log.Debug.Println("Additional slideshow data")
			return
		}

		// The data is JSON encoded.
		data, ok := s.Attr("data-lp-initial-images")
		if !ok {
			log.Debug.Println("No image data - data-lp-initial-images attr missing")
			return
		}

		err := json.NewDecoder(bytes.NewBufferString(data)).Decode(&scr.images)
		if err != nil {
			log.Debug.Printf("Error decoding image data: %v\n", err)
			return
		}
	})
}
