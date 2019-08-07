package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

// Merge all downloaded files into one
func mergeFiles(fileParts []string, folderPath string, fileName string) {

	mergedFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}

	defer mergedFile.Close()

	log.Printf("Start to merge files : [%s] ...\n", fileName)

	success := true
	for i := range fileParts {

		input, err := os.Open(fileParts[i])
		if err != nil {
			log.Println("Can't open file", err)
			success = false
			break
		}

		defer input.Close()

		_, err = io.Copy(mergedFile, input)
		if err != nil {
			log.Println("Copy Failed ...", err)
			success = false
			break
		}
	}

	// check if merged ..
	if success {
		log.Println("Success !!!  Deleting Tmp Folder ... ")
		defer func() {
			err := os.RemoveAll(folderPath)
			if err != nil {
				log.Println("Faile to remove folder ...", err)
			} else {
				log.Println("Folder Removed")
			}
		}()
	} else {
		log.Println("Merged failed ... Keep the files ... ")
	}
}

// Create a slice which contains path informaiton of each parts
func createParts(fileParts []string, partID uint64, folderPath string, fileName string, wg *sync.WaitGroup) {

	defer wg.Done()

	partPath := path.Join(folderPath, fileName)

	var builder strings.Builder
	fmt.Fprintf(&builder, "%s_%d_%d.part", partPath, partID, len(fileParts))

	fileParts[partID] = builder.String()

}

// Download file context in Range from start to end
func downloadRange(client *http.Client, url string, start uint64, end uint64, filePath string, wg *sync.WaitGroup) {

	defer wg.Done()

	log.Printf("file: [%s], start: %d, end : %d\n", filePath, start, end)

	// check if exist ...

	if prevFile, err := os.Stat(filePath); err == nil {
		// file exist , update progress
		start += uint64(prevFile.Size())
		log.Printf(" --- File: [%s] exists, update progress: %d\n", filePath, prevFile.Size())
	}

	if start > end {
		log.Printf("Already downloaded: [%s]\n", filePath)
		return
	}

	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Println("Failed to Create Request ...", err)
		return
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "bytes=%d-%d", start, end)
	request.Header.Set("Range", builder.String())

	response, err := client.Do(request)
	if err != nil {
		log.Println("HTTP Client Failed to execute ...", err)
		return
	}

	defer response.Body.Close()

	// test
	// fmt.Println(response)

	responseLength, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil || uint64(responseLength) != end-start+1 {
		log.Println("Content-Length mismatch ...", err)
		return
	}

	fileOut, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("File Open or Create Failed", err)
		return
	}

	defer fileOut.Close()

	_, err = io.Copy(fileOut, response.Body)
	if err != nil {
		log.Println("Failed to Write to file", err)
		return
	}

}

func downloadFile(url string, splitCount uint64, downloadWG *sync.WaitGroup) {

	defer downloadWG.Done()

	// TODO: convert to client with timeout ?
	response, err := http.Head(url)
	if err != nil {
		log.Println("Error getting response", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Printf("Status Code: %d, message: %s ... Skip for now", response.StatusCode, response.Status)
		return
	}

	// test: check header ...
	for key, values := range response.Header {
		for _, value := range values {
			fmt.Printf("%s : %s\n", key, value)
		}
	}

	// test: try to print body
	log.Println("Read Body ...")
	text, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading body", err)
		return
	}
	fmt.Println(text)

	// check if can split ...
	if response.Header.Get("Accept-Ranges") == "" || response.Header.Get("Accept-Ranges") == "none" {
		log.Println("Server does not accept range ...")
		splitCount = 1
	}

	// split it !
	length, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil {
		log.Println("Failed to convert Content-Length", err)
		return
	}

	sourceSize := uint64(length)
	fmt.Println(sourceSize)

	segmentSize := sourceSize / splitCount

	fileName := path.Base(url)
	folderName := tmpPrefix + fileName // TODO: add option for default path root

	fileParts := make([]string, splitCount)
	var fileWG sync.WaitGroup
	for i := range fileParts {
		fileWG.Add(1)
		go createParts(fileParts, uint64(i), folderName, fileName, &fileWG)
	}

	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		os.Mkdir(folderName, os.ModePerm)
	} else {
		log.Println("Tmp Folder Exists ... ")
	}

	fileWG.Wait()

	var wg sync.WaitGroup
	client := &http.Client{}
	for i := uint64(0); i < splitCount; i++ {

		startPos := i * segmentSize
		effectiveSize := segmentSize

		if i == splitCount-1 {
			effectiveSize += sourceSize % splitCount
		}

		endPos := startPos + effectiveSize - 1
		wg.Add(1)
		go downloadRange(client, url, startPos, endPos, fileParts[i], &wg)

	}

	wg.Wait()

	mergeFiles(fileParts, folderName, fileName)
}
