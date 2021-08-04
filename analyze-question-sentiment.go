// Before running this program, first fetch the data with fetch-all-questions
// into some base directory. Pass this base directory with the -dir flag to
// this program.
//
// To get a month-by-month breakdown from start date to end date, use the
// -bymonth flag.
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
	"path/filepath"
	"strings"
	"time"
)

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

type tagAnalysisResult struct {
	total             int
	negative          int
	closed            int
	closedAndNegative int
}

func mustParseTime(date string) time.Time {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func analyzeDir(baseDir string, tag string, fromDate time.Time, toDate time.Time) tagAnalysisResult {
	dirName := fmt.Sprintf("%s/%s", baseDir, tag)
	fileinfos, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}

	totalNum := 0
	numNegative := 0
	numClosed := 0
	numClosedAndNegative := 0

	for _, entry := range fileinfos {
		if strings.HasSuffix(entry.Name(), "json") {
			data, err := ioutil.ReadFile(filepath.Join(dirName, entry.Name()))
			if err != nil {
				log.Fatal(err)
			}

			var reply Reply
			if err = json.Unmarshal(data, &reply); err != nil {
				log.Fatal(err)
			}

			for _, item := range reply.Items {
				itemDate := time.Unix(int64(item.CreationDate), 0)
				if itemDate.Before(fromDate) || itemDate.After(toDate) {
					continue
				}

				totalNum++

				if item.Score < 0 {
					numNegative++
				}

				if item.ClosedDate > 0 {
					numClosed++

					if item.Score < 0 {
						numClosedAndNegative++
						//fmt.Println(item.Link, time.Unix(int64(item.CreationDate), 0), item.Score)
					}
				}
			}
		}
	}

	return tagAnalysisResult{
		total:             totalNum,
		negative:          numNegative,
		closed:            numClosed,
		closedAndNegative: numClosedAndNegative,
	}
}

func main() {
	dirFlag := flag.String("dir", "", "base directory to store results")
	fromDate := flag.String("fromdate", "", "start date in 2006-01-02 format")
	toDate := flag.String("todate", "", "end date in 2006-01-02 format")
	tagsFlag := flag.String("tags", "", "tags separated by commas")
	bymonthFlag := flag.Bool("bymonth", false, "analyze by month")

	flag.Parse()

	fDate := mustParseTime(*fromDate)
	tDate := mustParseTime(*toDate)
	tags := strings.Split(*tagsFlag, ",")

	fmt.Println(*dirFlag)

	emitResult := func(date time.Time, tr tagAnalysisResult) {
		negativeRatio := float64(tr.negative) / float64(tr.total)
		closedRatio := float64(tr.closed) / float64(tr.total)
		closedAndNegativeRatio := float64(tr.closedAndNegative) / float64(tr.total)
		fmt.Printf("%s,%d,%.3f,%.3f,%.3f\n", date.Format("2006-02-01"), tr.total, negativeRatio, closedRatio, closedAndNegativeRatio)
	}

	for _, tag := range tags {
		fmt.Printf("\n%s\n", tag)
		if *bymonthFlag {
			for d := fDate; d.Before(tDate); {
				endDate := d.AddDate(0, 1, 0)

				res := analyzeDir(*dirFlag, tag, d, endDate)
				emitResult(endDate, res)

				d = endDate
			}
		} else {
			res := analyzeDir(*dirFlag, tag, fDate, tDate)
			emitResult(tDate, res)
		}
	}
}
