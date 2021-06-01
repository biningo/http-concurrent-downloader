package main

import "log"

/**
*@Author lyer
*@Date 5/31/21 21:35
*@Describe
**/
//https://github.com/mojocn/flash
func main() {
	url := "https://download.jetbrains.com/go/goland-2020.2.2.dmg"
	downloader := NewDownloader(url, "/home/pb/Downloads", 1000, 5)
	if err := downloader.Download(); err != nil {
		log.Println(err)
	}
}
