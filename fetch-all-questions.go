// StackOverflow analysis using its API in Go.
//
// Registered on stackapps. Go to https://stackapps.com/apps/oauth/view/19622
// and log in, insert "key" for increased quota
//
// To get the increased quota, run with the env var STACK_KEY=<key>
//
// Eli Bendersky [https://eli.thegreenplace.net]
// This code is in the public domain.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Base query built with the explorer on
// https://api.stackexchange.com/docs/questions
//
// "https://api.stackexchange.com/2.2/questions?page=2&pagesize=100&fromdate=1610409600&todate=1613088000&order=desc&sort=activity&tagged=go&site=stackoverflow"

// Struct generated with https://mholt.github.io/json-to-go/
type Reply struct {
	Items []struct {
		Tags  []string `json:"tags"`
		Owner struct {
			Reputation   int    `json:"reputation"`
			UserID       int    `json:"user_id"`
			UserType     string `json:"user_type"`
			ProfileImage string `json:"profile_image"`
			DisplayName  string `json:"display_name"`
			Link         string `json:"link"`
		} `json:"owner"`
		IsAnswered       bool   `json:"is_answered"`
		ClosedDate       int64  `json:"closed_date"`
		ViewCount        int    `json:"view_count"`
		AcceptedAnswerID int    `json:"accepted_answer_id,omitempty"`
		AnswerCount      int    `json:"answer_count"`
		Score            int    `json:"score"`
		LastActivityDate int    `json:"last_activity_date"`
		CreationDate     int    `json:"creation_date"`
		LastEditDate     int    `json:"last_edit_date"`
		QuestionID       int    `json:"question_id"`
		ContentLicense   string `json:"content_license"`
		Link             string `json:"link"`
		Title            string `json:"title"`
	} `json:"items"`
	HasMore        bool `json:"has_more"`
	QuotaMax       int  `json:"quota_max"`
	QuotaRemaining int  `json:"quota_remaining"`
	Total          int  `json:"total"`
}

func makePageQuery(page int, tag string, fromDate time.Time, toDate time.Time) string {
	v := url.Values{}
	v.Set("page", strconv.Itoa(page))
	v.Set("pagesize", strconv.Itoa(100))
	v.Set("fromdate", strconv.FormatInt(fromDate.Unix(), 10))
	v.Set("todate", strconv.FormatInt(toDate.Unix(), 10))
	v.Set("order", "desc")
	v.Set("sort", "activity")
	v.Set("tagged", tag)
	v.Set("site", "stackoverflow")
	v.Set("key", os.Getenv("STACK_KEY"))
	return v.Encode()
}

func fetchResults(baseDir string, tags []string, fromDate time.Time, toDate time.Time) {
	for _, tag := range tags {
		dirName := fmt.Sprintf("%s/%s", baseDir, tag)
		os.RemoveAll(dirName)
		_ = os.Mkdir(dirName, 0777)

		fmt.Println("")
		fmt.Printf("Fetching tag '%s' to dir '%s'\n", tag, dirName)
		for page := 1; ; page++ {
			qs := makePageQuery(page, tag, fromDate, toDate)
			url := "https://api.stackexchange.com/2.2/questions?" + qs
			fmt.Println(url)

			resp, err := http.Get(url)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			fmt.Println("Response status:", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			pageFilename := fmt.Sprintf("%s/so%03d.json", dirName, page)
			err = ioutil.WriteFile(pageFilename, body, 0644)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Wrote", pageFilename)

			var reply Reply
			if err = json.Unmarshal(body, &reply); err != nil {
				log.Fatal(err)
			}

			if !reply.HasMore {
				break
			}
		}
	}
}

func mustParseTime(date string) time.Time {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func main() {
	dirFlag := flag.String("dir", "", "base directory to store results")
	fromDate := flag.String("fromdate", "", "start date in 2006-01-02 format")
	toDate := flag.String("todate", "", "end date in 2006-01-02 format")
	tagsFlag := flag.String("tags", "", "tags separated by commas")

	flag.Parse()

	fDate := mustParseTime(*fromDate)
	tDate := mustParseTime(*toDate)
	tags := strings.Split(*tagsFlag, ",")

	// Make dir / ignore error (if it exists, for example)
	_ = os.Mkdir(*dirFlag, 0777)
	fetchResults(*dirFlag, tags, fDate, tDate)
}
