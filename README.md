# lpscraper

A program for scraping images from the lonely planet website.

As of version 0.1, the program has many issues, but works for scraping images from a small collection of webpages, with some basic ability to handle timeouts.
The program crashes sometimes if a page cannot be loaded.

To try this out, clone this repository, and then:
 
 `go run lpscraper.go`
 
The images will appear in the directory lpscraper/data.
