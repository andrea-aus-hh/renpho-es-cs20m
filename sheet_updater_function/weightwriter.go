package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/sheets/v4"
)

func init() {
	functions.HTTP("WeightWriter", WeightWriter)
}

// WeightWriter is an HTTP Cloud Function with a request parameter.
func WeightWriter(w http.ResponseWriter, _ *http.Request) {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	spreadsheetId := "1WBAPDBnb01eJFBpBwmo_NDq2r9B44U0wyt7CZFxtus4"
	spreadsheet, err := srv.Spreadsheets.Get(spreadsheetId).Context(ctx).Do()
	if err != nil {
		log.Fatalf("unable to retrieve spreadsheet: %v", err)
	}
	dateToFind := time.Now().AddDate(0, 0, 1)

	candidateSheet := findCorrectSheet(spreadsheet, srv, spreadsheetId, dateToFind)

	log.Printf("Candidate sheet is called '%s'", candidateSheet.Properties.Title)

	readRange := fmt.Sprintf("%s!B:E", candidateSheet.Properties.Title)
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Printf("Unable to read data from sheet '%s': %v", candidateSheet.Properties.Title, err)
	}

	for i, row := range resp.Values {
		dateStr, ok := row[0].(string)
		if !ok {
			continue
		}

		dateParsed, err := time.Parse("02/01/2006", dateStr)
		if err != nil {
			continue
		}

		if datesAreEqual(dateParsed, dateToFind) {
			writeRange := candidateSheet.Properties.Title + "!E" + string(rune(i+1))
			valueRange := &sheets.ValueRange{
				Values: [][]interface{}{{"3,141592"}}, // Value to write
			}
			fmt.Fprintf(w, "Updating sheet '%s' at location '%s'\n", candidateSheet.Properties.Title, writeRange)
			_, err := srv.Spreadsheets.Values.Update(spreadsheetId, writeRange, valueRange).Do()
			if err != nil {
				log.Printf("Unable to update sheet '%s': %v", candidateSheet.Properties.Title, err)
				return
			} else {
				fmt.Fprintf(w, "Updated sheet '%s' at location '%s'\n", candidateSheet.Properties.Title, writeRange)
			}
			return
		}
	}
}

func findCorrectSheet(spreadsheet *sheets.Spreadsheet, srv *sheets.Service, spreadsheetId string, dateToFind time.Time) *sheets.Sheet {
	candidateSheet := spreadsheet.Sheets[0]
	for _, sheet := range spreadsheet.Sheets {
		if !strings.HasPrefix(sheet.Properties.Title, "Diario") {
			break
		}
		readRange := fmt.Sprintf("%s!B4", candidateSheet.Properties.Title)
		resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from sheet: %v", err)
		}
		firstDate, err := time.Parse("02/01/2006", resp.Values[0][0].(string))
		if err != nil {
			log.Fatalf("Unable to parse data from sheet: %v", err)
		}
		if firstDate.After(dateToFind) {
			break
		}
		candidateSheet = sheet
	}
	return candidateSheet
}

func datesAreEqual(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.Month() == date2.Month() && date1.Day() == date2.Day()
}
