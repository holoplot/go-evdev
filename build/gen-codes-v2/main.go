package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/holoplot/go-evdev/build/gen-codes-v2/parser"
)

const (
	eventCodes = "include/uapi/linux/input-event-codes.h"
	inputH     = "include/uapi/linux/input.h"
)

var urlBase = "https://raw.githubusercontent.com/torvalds/linux/%s/%s"

func downloadFileFromRepository(url string) string {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(data)
}

// fetchLinuxTags returns list of linux release tags, sorted from the oldest to the newest one
func fetchLinuxTags() (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("git", "ls-remote", "--sort=version:refname", "--tags", "https://github.com/torvalds/linux/")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

var tagRegex = regexp.MustCompile(`^\w{40}\W+refs/tags/(?P<version>v(\.?\d+)+)$`)

// parseTags processes raw tag list and returns tag versions only, skips release candidate versions,
// preserves incoming order
func parseTags(in string) []string {
	reader := strings.NewReader(in)
	scanner := bufio.NewScanner(reader)

	var tags = make([]string, 0)

	for scanner.Scan() {
		s := scanner.Text()

		if strings.HasSuffix(s, "^{}") {
			continue
		}

		match := tagRegex.FindStringSubmatch(s)

		if match == nil {
			continue
		}

		tags = append(tags, match[1])
	}

	if err := scanner.Err(); err != nil {
		return tags
	}

	return tags
}

func gitCommandAvailable() bool {
	err := exec.Command("git", "--version").Run()
	if err != nil {
		return false
	}
	return true
}

func main() {
	var disableComments, showTags bool
	var gitTag string

	flag.BoolVar(
		&disableComments, "disableCommentAutism", false,
		"disable including comments in the output file",
	)
	flag.StringVar(&gitTag, "tag", "", "select precise tag release of Linux")
	flag.BoolVar(&showTags, "tags", false, "list available Linux tag releases")
	flag.Parse()

	if !gitCommandAvailable() {
		fmt.Printf("git command is not available and it is required")
		os.Exit(1)
	}

	fmt.Println("Fetching tags...")
	err, stdout, stderr := fetchLinuxTags()
	if err != nil {
		fmt.Printf("err: %s\n", err)
		fmt.Printf("stderr: %s\n", stderr)
		os.Exit(1)
	}

	fmt.Println("Parsing tags...")
	tags := parseTags(stdout)
	if len(tags) == 0 {
		panic("no tags found")
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

	inputHURL := fmt.Sprintf(urlBase, selectedTag, inputH)
	eventCodesURL := fmt.Sprintf(urlBase, selectedTag, eventCodes)
	fmt.Println("Downloading files...")
	fmt.Printf("- %s\n- %s\n", inputHURL, eventCodesURL)
	inputHContent := downloadFileFromRepository(inputHURL)
	eventCodesContent := downloadFileFromRepository(eventCodesURL)

	c1 := parser.NewCodeProcessor(parser.SelectedPrefixesGroups["input.h"])
	c2 := parser.NewCodeProcessor(parser.SelectedPrefixesGroups["input-event-codes.h"])

	fmt.Println("Processing files...")
	elements1, err := c1.ProcessFile(strings.NewReader(inputHContent))
	if err != nil {
		fmt.Printf("processing file failed: %v\n", err)
		os.Exit(1)
	}

	elements2, err := c2.ProcessFile(strings.NewReader(eventCodesContent))
	if err != nil {
		fmt.Printf("processing file failed: %v\n", err)
		os.Exit(1)
	}

	allElements := append(elements2, elements1...)
	fmt.Println("Generating file...")
	data := parser.GenerateFile(allElements, disableComments, selectedTag, inputHURL, eventCodesURL)

	fmt.Println("Writing file...")
	fd, err := os.OpenFile("codes.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		panic(err)
	}
	_, err = fd.WriteString(data)
	if err != nil {
		panic(err)
	}

	fmt.Println("Done!")
}
