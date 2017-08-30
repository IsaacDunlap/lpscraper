package scraper

import (
	"compress/gzip"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lpscraper/log"
)

const (
	scheme                 = "https"
	lpDomain               = "www.lonelyplanet.com"
	DataDir                = "data"
	userAgent              = `Mozilla/5.0 (Windows NT 6.1; WOW64; rv:40.0) Gecko/20100101 Firefox/40.1`
	acceptEncoding         = "gzip, deflate, br"
	keepAlive              = "keep-alive"
	acceptLanguage         = "en-UK,en-US;q=0.8,en;q=0.6"
	accept                 = `text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8`
	upgradeInsecureReuests = "1"
)

const maxImageDownloadRetries = 5
const imageDonwloadRetryDelay = 10 * time.Second

var header = map[string]string{
	"Accept":          accept,
	"Accept-Encoding": acceptEncoding,
	"Accept-Language": acceptLanguage,
	"Connection":      keepAlive,
	"Host":            lpDomain,
	"Upgrade-Insecure-Requests": upgradeInsecureReuests,
	"User-Agent":                userAgent,
}

// The HTTP clients for the scraper
// TODO make safe for concurrent use
var client = &http.Client{}

// LpScraper scrapes a single lonely planet page for a country.
type LpScraper interface {
	// A unique identifier for this LpScraper
	// TODO make sure this is unique
	OID() string
	// Load queries the lonely planet webpage and returns the response.
	Load() (*http.Response, error)
	// Parse stores data for a lonely planet webpage into the scraper. Parse
	// doesn't close the response body - it is the caller's job to do
	// that.
	Parse(*http.Response) error
	// Download downloads any data from the webpage to the local machine.
	// It must be called after Parse.
	Download()
	Children() []LpScraper
}

func NewLpScraper(location string) LpScraper {
	scr := &lpScraper{}
	scr.url.Scheme = scheme
	scr.url.Host = lpDomain
	scr.url.Path = location

	log.Debug.Printf("Created new LpScraper for %s\n", scr.url.Path)
	return scr
}

// lpScraper is the implementation of the LpScraper interface.
type lpScraper struct {
	url    url.URL
	images []*Image
	pages  []string
}

type Image struct {
	URL   *url.URL
	Quote string
}

func (scr *lpScraper) OID() string {
	return scr.url.Path
}

func (scr *lpScraper) Children() []LpScraper {
	panic("implement me")
}

func (scr *lpScraper) Load() (*http.Response, error) {
	log.Debug.Printf("Loading %s", scr.url.String())

	req, err := http.NewRequest(http.MethodGet, scr.url.String(), nil)
	if err != nil {
		log.Debug.Printf("Failed to generate request for scraper for %s\n", scr.url.Path)
		return nil, err
	}

	req.Host = scr.url.Host
	for key, value := range header {
		req.Header.Add(key, value)
	}

	return client.Do(req)
}

func (scr *lpScraper) Download() {
	log.Debug.Printf("Downloading for %s\n", scr.url.String())

	for i, img := range scr.images {
		err := img.Download(i, scr.OID(), scr.url.String())
		placeName := strings.Replace(scr.OID(), string(filepath.Separator), "_", -1)
		if err != nil {
			log.Info.Printf(
				"Failed to download image at %s to %s\n",
				img.URL.String(),
				filepath.Join(DataDir, fmt.Sprintf("%s%04d.jpg", placeName, i)),
			)
			continue
		}
		log.Info.Printf(
			"Downloaded image at %s to %s\n",
			img.URL.String(),
			filepath.Join(DataDir, fmt.Sprintf("%s%04d.jpg", placeName, i)),
		)
		// TODO deal with image metadata and quote
	}
}

func (img Image) Download(suffix int, id string, referer string) error {

	var retries int
	var resp *http.Response

Retry:
	for ; retries < maxImageDownloadRetries; retries++ {
		req, err := http.NewRequest(http.MethodGet, img.URL.String(), nil)
		if err != nil {
			log.Debug.Printf("Failed to generate request for image %s\n", img.URL)
			return err
		}

		req.Host = img.URL.Host
		for key, value := range header {
			req.Header.Add(key, value)
		}
		req.Header.Add("referer", referer)
		resp, err = client.Do(req)
		if err != nil {
			log.Debug.Printf("No response for %s: %v\n", img.URL, err)
			return err
		}

		logResponse(resp)

		switch resp.StatusCode {
		case 200:
			// Response OK - stop retrying.
			break Retry
		case 504:
			// Tiemout - retry after a delay
			log.Debug.Printf("Server timeout for %s on attempt %d\n", img.URL, retries)
			resp.Body.Close()
			time.Sleep(imageDonwloadRetryDelay)
			continue
		default:
			// Cannot handle these codes. Quit.
			log.Debug.Printf("Bad response for %s\n", img.URL)
			resp.Body.Close()
			return errors.New("Bad response code")
		}
	}

	reader, err := newBodyReader(resp)
	if err != nil {
		log.Debug.Printf("Cannot read HTTP response for %s: %v\n", img.URL, err)
		return err
	}
	defer reader.Close()

	// Assume for now that the image is definitely a jpeg.
	if resp.Header.Get("content-type") != "image/jpeg" {
		log.Debug.Printf("No JPEG returned for %s\n", img.URL)
		return errors.New("Data not JPEG")
	}

	jpg, err := jpeg.Decode(reader)
	if err != nil {
		log.Debug.Printf("Cannot decode JPEG %s\n", img.URL)
		return err
	}

	// TODO in future have a function that produces a filename given an ID and suffix.
	placeName := strings.Replace(id, string(filepath.Separator), "_", -1)
	dst, err := os.Create(filepath.Join(DataDir, fmt.Sprintf("%s%04d.jpg", placeName, suffix)))
	if err != nil {
		log.Debug.Printf("Cannot create file for %s\n", img.URL)
		return err
	}

	err = jpeg.Encode(dst, jpg, &jpeg.Options{Quality: 100})
	if err != nil {
		log.Debug.Printf("Cannot write to file for %s\n", dst.Name())
		return err
	}

	if retries == maxImageDownloadRetries {
		return errors.New("Failed to download image due to timeout")
	}

	return nil
}

// Returns an appropriate reader to read the HTTP response body.
func newBodyReader(resp *http.Response) (io.ReadCloser, error) {
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		r, err := gzip.NewReader(resp.Body)
		return &bodyReader{r: r, contentEncoding: "gzip"}, err
	default:
		return &bodyReader{r: resp.Body, contentEncoding: ""}, nil
	}
}

type bodyReader struct {
	r               io.ReadCloser
	contentEncoding string
}

func (r *bodyReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

// Closes the underlying reader - we only close a reader that
// wraps the response body's reader. Otherwise don't close it -
// the user of the lpScraper will do that.
func (r *bodyReader) Close() error {
	switch r.contentEncoding {
	case "":
		return nil
	case "gzip":
		return r.r.Close()
	default:
		panic("Unreachable")
	}
}

func logResponse(resp *http.Response) {
	// Log details of request and response
	log.Debug.Printf("Request method %s\n", resp.Request.Method)
	log.Debug.Printf("Request URL %s\n", resp.Request.URL.String())
	log.Debug.Printf("Request host %s\n", resp.Request.Host)
	for key, value := range resp.Request.Header {
		log.Debug.Printf("Request header %s: %s\n", key, value)
	}
	log.Debug.Printf("Response status %s\n", resp.Status)
	log.Debug.Printf("Response content length %d\n", resp.ContentLength)
	log.Debug.Printf("Response transfer encoding %s\n", resp.TransferEncoding)
	for key, value := range resp.Header {
		log.Debug.Printf("Response header %s: %s\n", key, value)
	}
}
