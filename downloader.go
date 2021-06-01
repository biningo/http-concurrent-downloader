package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
*@Author lyer
*@Date 6/1/21 10:14
*@Describe
**/
// Downloader http range downloader.
type Downloader struct {
	ContentLength int           // Content-Length
	HttpURL       string        // download url
	FilePath      string        // save file path
	FileName      string        // file name
	parts         []ContentPart // every parts
	TotalPart     int
	RetryCount    int // download retry count
}

// ContentPart file part
type ContentPart struct {
	Index      int    // parts index
	Start      int    // range start
	End        int    // range end
	Data       []byte // range content data
	Successful bool   // download successful
}

func NewDownloader(url string, path string, totalPart int, retryCount int) *Downloader {
	return &Downloader{
		HttpURL:    url,
		FilePath:   path,
		TotalPart:  totalPart,
		RetryCount: retryCount,
	}
}

//Download download file
func (d *Downloader) Download() (err error) {

	defer func() {
		if ie := recover(); ie != nil {
			err = ie.(error)
		}
	}()

	e := d.verifyHeader()
	checkError(e)

	e = d.parseHeader()
	checkError(e)

	d.parts = make([]ContentPart, d.TotalPart)
	d.initPartRange()

	wg := sync.WaitGroup{}
	wg.Add(d.TotalPart)
	for i := 0; i < d.TotalPart; i++ {
		go func(index int) {
			defer wg.Done()
			var errDownload error
			for j := 0; j < d.RetryCount; j++ {
				if errDownload = d.downloadPart(index); errDownload == nil {
					d.parts[index].Successful = true
					log.Println(index, "is ok!")
					return
				}
				log.Println(index, errDownload)
			}
		}(i)
	}
	wg.Wait()
	if successful := d.checkDownloadSuccessful(); !successful {
		return errors.New("download failure")
	}
	e = d.mergeParts()
	checkError(e)
	return
}

func (d *Downloader) verifyHeader() error {
	resp, err := http.DefaultClient.Head(d.HttpURL)
	if err != nil {
		return err
	}
	acceptRange := resp.Header.Get("Accept-Ranges")
	if strings.ToLower(acceptRange) != "bytes" {
		return errors.New("range requests are not supported")
	}
	return nil
}

func (d *Downloader) parseHeader() (err error) {
	defer func() {
		if ierr := recover(); ierr != nil {
			err = ierr.(error)
		}
	}()

	resp, e := http.DefaultClient.Head(d.HttpURL)
	checkError(e)

	d.ContentLength, err = strconv.Atoi(resp.Header.Get("Content-Length"))
	checkError(e)

	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition == "" {
		d.FileName = time.Now().Format("2006-01-02")
		return nil
	}
	_, params, e := mime.ParseMediaType(contentDisposition)
	checkError(e)
	d.FileName = params["filename"]
	return nil

}

func (d *Downloader) initPartRange() {
	size := d.ContentLength / d.TotalPart
	for i := range d.parts {
		d.parts[i].Index = i
		if i == 0 {
			d.parts[i].Start = 0
		} else {
			d.parts[i].Start = d.parts[i-1].End + 1
		}
		if i == d.TotalPart-1 {
			d.parts[i].End = d.ContentLength - 1
		} else {
			d.parts[i].End = d.parts[i].Start + size
		}
	}
}

func (d *Downloader) checkDownloadSuccessful() bool {
	for _, p := range d.parts {
		if !p.Successful {
			return false
		}
	}
	return true
}

func (d *Downloader) mergeParts() (err error) {
	defer func() {
		if ierr := recover(); ierr != nil {
			err = ierr.(error)
		}
	}()

	file, e := os.Create(filepath.Join(d.FilePath, d.FileName))
	checkError(e)
	defer func() {
		if e := file.Close(); e != nil {
			log.Println(e)
		}
	}()

	for _, part := range d.parts {
		_, e = file.Write(part.Data)
		checkError(e)
	}
	return nil
}

func (d *Downloader) downloadPart(index int) (err error) {
	defer func() {
		if ie := recover(); ie != nil {
			err = ie.(error)
		}
	}()

	req, e := http.NewRequest(http.MethodGet, d.HttpURL, nil)
	checkError(e)

	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", d.parts[index].Start, d.parts[index].End))
	resp, e := http.DefaultClient.Do(req)
	checkError(e)

	d.parts[index].Data, e = io.ReadAll(resp.Body)
	checkError(e)

	return
}
