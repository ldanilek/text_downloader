package main

import (
	"os"
	"io"
	"fmt"
	"net/http"
	"encoding/csv"
	"sync"
	"errors"
	"bufio"
	"strings"
	"net/url"
	"regexp"
	"path/filepath"
)

func retry(f func() error) {
	for {
		err := f()
		if err == nil {
			return
		}
		fmt.Fprintf(os.Stderr, "-------------------------\n" +
			"ERROR\n" +
			"-------------------------\n" +
			"%v\n",
			err,
		)
	}
}

func fetchURL(url string) io.ReadCloser {
	var resp *http.Response
	retry(func() error {
		var err error
		resp, err = http.Get(url)
		return err
	})
	return resp.Body
}

func readFile(path string) io.ReadCloser {
	var file *os.File
	retry(func() error {
		var err error
		file, err = os.Open(path)
		return err
	})
	return file
}

func writeFile(toPath string) io.WriteCloser {
	var file *os.File
	retry(func() error {
		var err error
		file, err = os.OpenFile(toPath, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0660)
		return err
	})
	return file
}

type Textbook struct {
	title string
	author string
	electronicISBN string
	doiURL string
	openURL string
}

func readCSV(path string, output chan<- Textbook) error {
	file := readFile(path)
	defer file.Close()
	csvReader := csv.NewReader(file)
	rowIndex := 0
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return errors.New(fmt.Sprintf("failed to read row %d: %v", rowIndex, err))
		}
		if len(record) != 22 {
			return errors.New(fmt.Sprintf("Unexpected number of columns in row %d: %v", rowIndex, record))
		}
		if rowIndex == 0 {
			// The first line of CSV is column names, not a valid row.
			rowIndex++
			continue
		}
		output <- Textbook{
			title: record[0],
			author: record[1],
			electronicISBN: record[7],
			doiURL: record[17],
			openURL: record[18],
		}
		rowIndex++
	}
}

var re = regexp.MustCompile("a href=\"(.*)\" title=\"Download this book in PDF format\"")

func contentURL(textbook Textbook) string {
	var urlForContent string
	retry(func() error {
		landingPageReader := fetchURL(textbook.openURL)
		defer landingPageReader.Close()
		lineReader := bufio.NewReader(landingPageReader)
		for {
			line, err := lineReader.ReadString('\n')
			if err != nil {
				return err
			}
			if strings.Contains(line, "Download this book in PDF format") {
				submatch := re.FindStringSubmatch(line)
				if len(submatch) < 2 {
					return errors.New(fmt.Sprintf("couldn't parse content url from download link %s for textbook %v", line, textbook))
				}
				urlPath := submatch[1]
				parsedURL, err := url.Parse(textbook.openURL)
				if err != nil {
					return err
				}
				parsedURL.Path = urlPath
				parsedURL.RawQuery = ""
				urlForContent = parsedURL.String()
				return nil
			}
		}
		return errors.New(fmt.Sprintf("couldn't find pdf download link for textbook %v", textbook))
	})	
	return urlForContent
}

func downloadContent(url string, toPath string) {
	retry(func() error {
		var retErr error
		pipeReader, pipeWriter := io.Pipe()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer pipeWriter.Close()
			urlReader := fetchURL(url)
			defer urlReader.Close()
			_, err := io.Copy(pipeWriter, urlReader)
			pipeWriter.CloseWithError(err)
		}()
		go func() {
			defer wg.Done()
			defer pipeReader.Close()
			fileWriter := writeFile(toPath)
			defer fileWriter.Close()
			_, retErr = io.Copy(fileWriter, pipeReader)
		}()
		wg.Wait()
		return retErr
	})
}

func sanitizePath(path string) string {
	return strings.ReplaceAll(path, "/", "_")
}

func processTextbook(textbook Textbook) {
	url := contentURL(textbook)
	filename := sanitizePath(fmt.Sprintf("%s (%s).pdf", textbook.title, textbook.electronicISBN))
	toPath := filepath.Join("output", filename)
	downloadContent(url, toPath)
	fmt.Printf("Downloaded %s\n", filename)
}

func processTextbooks(textbooks <-chan Textbook) {
	for textbook := range textbooks {
		processTextbook(textbook)
	}
}

func downloadTextbooksFromFile(path string) error {
	textbooks := make(chan Textbook)
	var err error
	var wg sync.WaitGroup
	workerPoolSize := 100
	wg.Add(workerPoolSize+1)
	go func() {
		defer wg.Done()
		defer close(textbooks)
		err = readCSV(path, textbooks)
	}()
	for worker := 0; worker < workerPoolSize; worker++ {
		go func() {
			defer wg.Done()
			processTextbooks(textbooks)
		}()
	}
	wg.Wait()
	return err
}

func main() {
	retry(func() error {
		return downloadTextbooksFromFile("Free+English+textbooks.csv")
	})
}
