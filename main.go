package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {

	filePtr := flag.String("f", "", "Add file containing URL's")
	timeOutPtr := flag.Int("t", 10, "Add timeout in seconds")
	concurrentPtr := flag.Int("c", 10, "Add concurrent checks in units")

	flag.Parse()

	if *filePtr == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	domains, totalCount := read(*filePtr)
	result := checker(domains, timeOutPtr, concurrentPtr)

	var goodCounter, badCounter int
	var badStatus []string

	start := time.Now()
	for status := range result {
		if strings.HasPrefix(status, "200") {
			fmt.Println(status)
			goodCounter++
		} else {
			badStatus = append(badStatus, status)
			badCounter++
		}
	}
	timeElapsed := time.Since(start)

	for _, status := range badStatus {
		fmt.Println(status)
	}

	fmt.Printf("Checked %v domains (succes:%v bad:%v) - took: %v \n",
		totalCount,
		goodCounter,
		badCounter,
		timeElapsed,
	)
}

func read(file string) ([]string, int) {
	var count int
	domains := []string{}

	fileHandle, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer fileHandle.Close()

	scanner := bufio.NewScanner(fileHandle)
	for scanner.Scan() {
		domains = append(domains, scanner.Text())
		count++
	}

	return domains, count
}

func checker(domains []string, timeOutPtr, concurrentPtr *int) chan string {
	out := make(chan string)
	done := make(chan bool)
	sem := make(chan bool, *concurrentPtr)

	check := func(domain string) {
		sem <- true
		defer func() { <-sem }()

		client := http.Client{
			Timeout: time.Duration(time.Duration(*timeOutPtr) * time.Second),
		}

		req, _ := http.NewRequest("GET", fmt.Sprintf("http://www.%v", domain), nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36")

		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start)

		if err != nil {
			out <- fmt.Sprintf("ERR, %v, Error: %v", domain, err)
			done <- true
		} else {
			out <- fmt.Sprintf("%v, %v, took: %v", resp.Status, domain, elapsed)
			done <- true
		}
	}

	for _, v := range domains {
		go check(v)
	}

	go func() {
		for i := 0; i < len(domains); i++ {
			<-done
		}
		close(out)
	}()

	return out
}
