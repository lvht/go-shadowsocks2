package main

import (
	"bufio"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var sources = []string{
	"https://raw.githubusercontent.com/gfwlist/gfwlist/master/gfwlist.txt",
	"https://bitbucket.org/gfwlist/gfwlist/raw/HEAD/gfwlist.txt",
	"https://gitlab.com/gfwlist/gfwlist/raw/master/gfwlist.txt",
}

// FetchBlockedAddrs download blocked addrs from the gfwlist
func FetchBlockedAddrs() ([]string, error) {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	var err error
	var resp *http.Response
	for _, url := range sources {
		if resp, err = client.Get(url); err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := base64.NewDecoder(base64.StdEncoding, resp.Body)
	scanner := bufio.NewScanner(decoder)

	res := []*regexp.Regexp{
		regexp.MustCompile("^\\|+"),
		regexp.MustCompile("https?://"),
		regexp.MustCompile("^\\."),
		regexp.MustCompile("^\\*.*?\\."),
	}

	scanner.Scan() // skip [AutoProxy 0.2.9]
	var addrs = []string{}
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '!' || line[0] == '@' {
			continue
		}
		if strings.IndexByte(line, '/') > 0 {
			continue
		}
		if strings.IndexByte(line, '*') > 0 {
			continue
		}

		for _, re := range res {
			line = re.ReplaceAllString(line, "")
		}
		if strings.IndexByte(line, '.') > 0 {
			addrs = append(addrs, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return addrs, nil
}
