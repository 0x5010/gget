package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/0x5010/pf"
)

var (
	defaultClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	defaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/68.0.3440.106 Safari/537.36"
)

type HttpDownloader struct {
	url         string
	pf          *pf.PF
	concurrency int
	ua          string
}

func NewHttpDownloader(url, ua string, concurrency int, piecesize int64) *HttpDownloader {
	var err error
	if ua == "" {
		ua = defaultUA
	}
	req, err := http.NewRequest("GET", url, nil)
	Panic(err)

	req.Header.Add("User-Agent", ua)
	resp, err := defaultClient.Do(req)
	Panic(err)

	if resp.Header.Get("Accept-Ranges") == "" {
		fmt.Println("Target url is not supported range download, fallback to concurrency 1")
		concurrency = 1
	}
	cl := resp.Header.Get("Content-Length")

	size := int64(1)
	if cl != "" {
		size, err = strconv.ParseInt(cl, 10, 64)
		Panic(err)
		fmt.Printf("Download target size: %.1f MB\n", float64(size)/(1024*1024))
	} else {
		concurrency = 1
		size = 0
	}
	if size == 0 {
		piecesize = 0
	} else if piecesize == 0 {
		piecesize = min(65536, (size+int64(concurrency)-1)/int64(concurrency))
	}

	filename := filepath.Base(url)
	opts := []pf.PFOption{}
	if piecesize != 0 {
		opts = append(opts, pf.SetPieceSize(piecesize))
	}

	f, err := pf.New(filename, size, opts...)
	return &HttpDownloader{
		url:         url,
		ua:          ua,
		pf:          f,
		concurrency: concurrency,
	}
}

func (hd *HttpDownloader) Run() {
	var wg sync.WaitGroup
	for i := 0; i < hd.concurrency; i++ {
		wg.Add(1)
		go func() {
			for {
				index := hd.pf.Progress.FindFirstClear()
				if index == -1 {
					break
				}
				err := hd.downPiece(index)
				if err != nil {
					fmt.Println(err)
					hd.pf.Progress.Clear(index)
					time.Sleep(1 * time.Second)
					continue
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	hd.pf.WaitFinish()
}

func (hd *HttpDownloader) downPiece(index int) error {
	req, err := http.NewRequest("GET", hd.url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("User-Agent", hd.ua)

	start := int64(index) * hd.pf.PieceSize
	end := start + hd.pf.PieceSize - 1
	req.Header.Set("Range", "bytes="+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10))

	res, err := defaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 && res.StatusCode != 206 {
		return fmt.Errorf("status error")
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return hd.pf.Write(index, data)
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func Panic(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
