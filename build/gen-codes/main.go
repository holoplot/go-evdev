package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/holoplot/go-evdev/build/gen-codes/parser"
)

const (
	eventCodes = "include/uapi/linux/input-event-codes.h"
	inputH     = "include/uapi/linux/input.h"
)

func downloadFile(url string) []byte {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Get(url)
	if err != nil {
		panic(fmt.Errorf("downlload \"%s\" file failed: %v", url, err))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("reading data of \"%s\" file failed: %v", url, err))
	}

	return data
}

type Tag struct {
	Ref string `json:"ref"`
}

func getTags() []string {
	var tagRegex = regexp.MustCompile(`^refs/tags/(?P<version>v(\.?\d+)+)$`)
	var tags []Tag

	url := "https://api.github.com/repos/torvalds/linux/git/refs/tags"
	err := json.Unmarshal(downloadFile(url), &tags)
	if err != nil {
		panic(fmt.Errorf("failed to parse json \"%s\" file: %v", url, err))
	}

	var tagsString []string

	for _, t := range tags {
		match := tagRegex.FindStringSubmatch(t.Ref)
		if match == nil {
			continue
		}

		tagsString = append(tagsString, match[1])
	}

	return tagsString
}

func main() {
	var disableComments, showTags bool
	var gitTag string

	flag.BoolVar(
		&disableComments, "disableComments", false,
		"disable including comments in the output file",
	)
	flag.StringVar(&gitTag, "tag", "", "select precise tag release of Linux")
	flag.BoolVar(&showTags, "tags", false, "list available Linux tag releases")
	flag.Parse()

	fmt.Println("Fetching tags...")
	tags := getTags()

	if len(tags) == 0 {
		fmt.Printf("no tags found\n")
		os.Exit(1)
	}

	newestTag := tags[len(tags)-1]

	fmt.Printf("The newest available tag: %s\n", newestTag)
	if showTags {
		for _, tag := range tags {
			fmt.Printf("- %s\n", tag)
		}
		os.Exit(0)
	}

	var selectedTag = newestTag
	if gitTag != "" {
		var found bool
		for _, tag := range tags {
			if tag == gitTag {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("tag \"%s\" does not exist\n", gitTag)
			os.Exit(1)
		}
		selectedTag = gitTag
		fmt.Printf("found \"%s\" tag\n", gitTag)
		fmt.Printf("Warning! This codegen was not tested with heavily outdated releases\n")
		fmt.Printf("nor the code that depends on it.\n")
	}

	urlBase := "https://raw.githubusercontent.com/torvalds/linux/%s/%s"
	inputHURL := fmt.Sprintf(urlBase, selectedTag, inputH)
	eventCodesURL := fmt.Sprintf(urlBase, selectedTag, eventCodes)

	fmt.Println("Downloading files...")
	fmt.Printf("- %s\n- %s\n", inputHURL, eventCodesURL)
	inputHContent := string(downloadFile(inputHURL))
	eventCodesContent := string(downloadFile(eventCodesURL))

	c1 := parser.NewCodeProcessor(parser.SelectedPrefixesGroups["input.h"])
	c2 := parser.NewCodeProcessor(parser.SelectedPrefixesGroups["input-event-codes.h"])

	fmt.Println("Processing files...")

	elements1, err := c1.ProcessFile(strings.NewReader(inputHContent))
	if err != nil {
		fmt.Printf("processing input.h file failed: %v\n", err)
		os.Exit(1)
	}
	elements2, err := c2.ProcessFile(strings.NewReader(eventCodesContent))
	if err != nil {
		fmt.Printf("processingi nput-event-codes.h file failed: %v\n", err)
		os.Exit(1)
	}

	allElements := append(elements2, elements1...)

	fmt.Println("Generating file...")
	data := parser.GenerateFile(allElements, disableComments, selectedTag, inputHURL, eventCodesURL)

	formatted, err := format.Source([]byte(data))
	if err != nil {
		fmt.Printf("Cannot format: %s", err)
		os.Exit(1)
	}
	fileName := "codes.go"
	if err := os.WriteFile(fileName, formatted, 0o644); err != nil {
		fmt.Printf("Failed to write data to %s file: %v", fileName, err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}
