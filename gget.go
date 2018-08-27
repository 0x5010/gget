package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

var (
	h           bool
	url         string
	concurrency int
	userAgent   string
)

func main() {
	flag.IntVar(&concurrency, "c", runtime.NumCPU(), "Number of multiple requests to make at a time")
	flag.StringVar(&userAgent, "A", "", "Set User-Agent STRING")
	flag.BoolVar(&h, "h", false, "help")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 || concurrency == 0 || h {
		usage()
	}
	url = args[0]
	if userAgent == "" {
		userAgent = defaultUA
	}

	hd := NewHttpDownloader(url, userAgent, concurrency, 0)
	hd.Run()
}

func usage() {
	fmt.Println("gget [URL]")
	flag.PrintDefaults()
	os.Exit(0)
}
