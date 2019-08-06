package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
)

const tmpPrefix string = "tmp_"
const defaultSplitCount uint64 = 30

// const defaultInput string = "input.txt"

const defaultTags = "edit|save|view|video|download|file"

func main() {
	fmt.Println("GoFetch - multi-downloader")

	// add server support

	var validPath = regexp.MustCompile("^/(" + defaultTags + ")/([a-zA-Z0-9_-]+)$")
	var validResource = regexp.MustCompile("^/(resource)/([a-zA-Z0-9_-]+\\.mp4)$")

	http.HandleFunc("/view/", makeHandler(viewHandler, validPath))
	http.HandleFunc("/edit/", makeHandler(editHandler, validPath))
	http.HandleFunc("/save/", makeHandler(saveHandler, validPath))
	http.HandleFunc("/video/", makeHandler(videoHandler, validPath))
	http.HandleFunc("/download/", makeHandler(downloadHandler, validPath))

	http.HandleFunc("/file/", makeHandler(fileHandler, validPath))

	http.HandleFunc("/resource/", makeHandler(resourceHandler, validResource))

	log.Fatal(http.ListenAndServe(":8051", nil))

}
