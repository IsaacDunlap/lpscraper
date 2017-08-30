package log

import (
	"log"
	"os"
	"runtime"
	"path/filepath"
)

const LogFileName = "log.txt"

var (
	Info      *log.Logger
	Debug     *log.Logger
	File      *os.File
)

func init() {
	_, thisFile, _, ok := runtime.Caller(1)
	if !ok {
		log.Fatal("Could not identify current file")
	}

	File, err := os.Create(filepath.Join(filepath.Dir(thisFile), LogFileName))
	if err != nil {
		log.Fatal(err)
	}

	Info = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)
	Debug = log.New(File, "DEBUG: ", log.LstdFlags|log.Lshortfile)
}
