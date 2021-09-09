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
	"database/sql"
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

	// min and max dates of actual items
	minDate sql.NullTime
	maxDate sql.NullTime
}

func mustParseDate(date string) time.Time {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func parseDate(date string) sql.NullTime {
	t, err := time.Parse("2006-01-02", date)
	ok := err == nil
	return sql.NullTime{
		Valid: ok,
		Time:  t,
	}
}

// analyzeDir analyzes the question data in base directory baseDir for the given
// tag. If fromDate and toDate are provided, then only questions between fromDate
// and toDate (inclusive) are considered.
func analyzeDir(baseDir string, tag string, fromDate sql.NullTime, toDate sql.NullTime) tagAnalysisResult {
	dirName := fmt.Sprintf("%s/%s", baseDir, tag)
	fileinfos, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}

	var tr tagAnalysisResult

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
				if fromDate.Valid && itemDate.Before(fromDate.Time) {
					continue
				}
				if toDate.Valid && itemDate.After(toDate.Time) {
					continue
				}

				tr.total++

				if item.Score < 0 {
					tr.negative++
				}

				if item.ClosedDate > 0 {
					tr.closed++

					if item.Score < 0 {
						tr.closedAndNegative++
						//fmt.Println(item.Link, time.Unix(int64(item.CreationDate), 0), item.Score)
					}
				}

				if !tr.minDate.Valid || itemDate.Before(tr.minDate.Time) {
					tr.minDate.Time = itemDate
					tr.minDate.Valid = true
				}
				if !tr.maxDate.Valid || itemDate.After(tr.maxDate.Time) {
					tr.maxDate.Time = itemDate
					tr.maxDate.Valid = true
				}
			}
		}
	}
	return tr
}

func main() {
	dirFlag := flag.String("dir", "", "base directory with results")
	fromDate := flag.String("fromdate", "", "start date in 2006-01-02 format")
	toDate := flag.String("todate", "", "end date in 2006-01-02 format")
	tagsFlag := flag.String("tags", "", "tags separated by commas")
	bymonthFlag := flag.Bool("bymonth", false, "analyze by month")

	flag.Parse()

	fDate := parseDate(*fromDate)
	tDate := parseDate(*toDate)
	tags := strings.Split(*tagsFlag, ",")

	emitResult := func(date sql.NullTime, tr tagAnalysisResult) {
		negativeRatio := float64(tr.negative) / float64(tr.total)
		closedRatio := float64(tr.closed) / float64(tr.total)
		closedAndNegativeRatio := float64(tr.closedAndNegative) / float64(tr.total)

		if !date.Valid {
			// if not explicit date, consider the max encountered date
			date = tr.maxDate
		}

		fmt.Printf("%s,%d,%.3f,%.3f,%.3f\n", date.Time.Format("2006-01-02"), tr.total, negativeRatio, closedRatio, closedAndNegativeRatio)
	}

	for _, tag := range tags {
		fmt.Printf("\n%s\n", tag)
		if *bymonthFlag {
			if !fDate.Valid || !tDate.Valid {
				log.Fatal("-bymonth requires -fromdate and -todate, for now")
			}
			for d := fDate; d.Time.Before(tDate.Time); {
				endDate := sql.NullTime{
					Valid: true,
					Time:  d.Time.AddDate(0, 1, 0), // add a month
				}

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
