package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"sync"
)

const tmpPrefix string = "tmp_"
const defaultSplitCount uint64 = 30
const defaultInput string = "input.txt"

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
		log.Println("Success !!!  Delete Tmp Folder ... ")
		defer os.RemoveAll(folderPath)
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

func main() {
	fmt.Println("GoFetch - multi-downloader")

	fmt.Println("Read input from file")

	inputFile, err := os.Open(defaultInput)
	if err != nil {
		log.Fatal("Failed to open input file\n")
	}

	defer inputFile.Close()

	// scanner := bufio.NewScanner(os.Stdin)
	scanner := bufio.NewScanner(inputFile)

	var wg sync.WaitGroup
	for scanner.Scan() {
		targetURL := scanner.Text()

		log.Printf("Start Downloading :[%s] ...\n", targetURL)

		wg.Add(1)
		go downloadFile(targetURL, defaultSplitCount, &wg)

	}

	wg.Wait()

}
