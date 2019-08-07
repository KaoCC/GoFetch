package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"text/template"
	"time"
)

// Page ... Title: the title of web page. Body: the contend. Resource: the related links.
type Page struct {
	Title    string
	Body     []byte
	Resource []string
}

var renderTemplate = makeRenderer()

func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func videoHandler(w http.ResponseWriter, r *http.Request, title string) {

	p := &Page{Title: title, Resource: []string{title + ".mp4"}}

	renderTemplate(w, "video", p)
}

func resourceHandler(w http.ResponseWriter, r *http.Request, resource string) {

	res, err := os.Open(resource)
	if err != nil {
		log.Println(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Close()

	http.ServeContent(w, r, resource, time.Now(), res)
}

func fileHandler(w http.ResponseWriter, r *http.Request, title string) {

	matches, _ := filepath.Glob("*.mp4")

	p := &Page{Title: title, Resource: matches}

	renderTemplate(w, "file", p)
}

// TODO: move the download logic to a spearate process or thread
func downloadHandler(w http.ResponseWriter, r *http.Request, title string) {

	input := title + ".txt"

	inputFile, err := os.Open(input)
	if err != nil {
		log.Fatal("Failed to open input file\n")
	}

	defer inputFile.Close()

	// scanner := bufio.NewScanner(os.Stdin)
	scanner := bufio.NewScanner(inputFile)

	var wg sync.WaitGroup
	for scanner.Scan() {
		targetURL := scanner.Text()

		// check if exist ?

		fileName := path.Base(targetURL)
		if _, err := os.Stat(fileName); err == nil {
			log.Printf("File [%s] exists, skip for now ...\n", fileName)
			continue
		}

		log.Printf("Start Downloading :[%s] ...\n", targetURL)

		wg.Add(1)
		go downloadFile(targetURL, defaultSplitCount, &wg)

	}

	wg.Wait()

	// show page ?

	http.Redirect(w, r, "/file/"+title, http.StatusFound)

}

func makeRenderer() func(w http.ResponseWriter, tmpl string, p *Page) {

	var templates = template.Must(template.ParseFiles("edit.html", "view.html", "video.html", "file.html"))

	return func(w http.ResponseWriter, tmpl string, p *Page) {
		err := templates.ExecuteTemplate(w, tmpl+".html", p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string), pattern *regexp.Regexp) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := pattern.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}
