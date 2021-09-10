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
	"log"
	"os"
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
	minDate time.Time
	maxDate time.Time
}

func parseDate(date string) time.Time {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{} // zero, here means undefined
	}
	return t
}

// analyzeDir analyzes the question data in base directory baseDir for the given
// tag. If fromDate and toDate are non-zero, then only questions between fromDate
// and toDate (inclusive) are considered.
func analyzeDir(baseDir string, tag string, fromDate time.Time, toDate time.Time) tagAnalysisResult {
	dirName := fmt.Sprintf("%s/%s", baseDir, tag)
	entries, err := os.ReadDir(dirName)
	failonf(err, "reading directory %q", dirName)

	var tr tagAnalysisResult

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "json") {
			path := filepath.Join(dirName, entry.Name())
			data, err := os.ReadFile(path)
			failonf(err, "reading file %q", path)

			var reply Reply
			err = json.Unmarshal(data, &reply)
			failonf(err, "unmarshalling")

			for _, item := range reply.Items {
				itemDate := time.Unix(int64(item.CreationDate), 0)
				if !fromDate.IsZero() && itemDate.Before(fromDate) {
					continue
				}
				if !toDate.IsZero() && itemDate.After(toDate) {
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

				if tr.minDate.IsZero() || itemDate.Before(tr.minDate) {
					tr.minDate = itemDate
				}
				if tr.maxDate.IsZero() || itemDate.After(tr.maxDate) {
					tr.maxDate = itemDate
				}
			}
		}
	}
	return tr
}

// readFolderNames discovers and returns the names of the subfolders
// inside dir (non-recursively).
func readFolderNames(dirpath string) []string {
	entries, err := os.ReadDir(dirpath)
	failonf(err, "reading directory %q", dirpath)

	var folders []string
	for _, entry := range entries {
		if entry.IsDir() {
			folders = append(folders, entry.Name())
		}
	}
	return folders
}

// failonf exits with a message if err is not nil.
func failonf(err error, pattern string, args ...interface{}) {
	if err != nil {
		log.Println(err)
		log.Fatalf(pattern, args...)
	}
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

	if len(*dirFlag) == 0 {
		log.Fatal("-dir must be provided and cannot be empty. Please use the folder where the data was fetched.")
	}

	emitResult := func(date time.Time, tr tagAnalysisResult) {
		negativeRatio := float64(tr.negative) / float64(tr.total)
		closedRatio := float64(tr.closed) / float64(tr.total)
		closedAndNegativeRatio := float64(tr.closedAndNegative) / float64(tr.total)

		if date.IsZero() {
			// if not explicit date, consider the max encountered date
			date = tr.maxDate
		}

		fmt.Printf("%s,%d,%.3f,%.3f,%.3f\n", date.Format("2006-01-02"), tr.total, negativeRatio, closedRatio, closedAndNegativeRatio)
	}

	if *tagsFlag == "" {
		// No explicit tags specified by user => then discover
		// the subfolders of the results base directory
		tags = readFolderNames(*dirFlag)
	}

	for _, tag := range tags {
		fmt.Printf("\n%s\n", tag)
		if *bymonthFlag {
			if fDate.IsZero() || tDate.IsZero() {
				log.Fatal("-bymonth requires -fromdate and -todate, for now")
			}
			for d := fDate; d.Before(tDate); {
				endDate := d.AddDate(0, 1, 0) // add a month

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
