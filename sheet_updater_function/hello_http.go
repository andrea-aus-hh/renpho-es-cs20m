package sheet_updater_function

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/sheets/v4"
)

func init() {
	functions.HTTP("HelloHTTP", HelloHTTP)
}

// HelloHTTP is an HTTP Cloud Function with a request parameter.
func HelloHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Hello HTTP!\n")
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
	dateToFind := time.Date(2025, time.January, 10, 0, 0, 0, 0, time.UTC)
	for _, sheet := range spreadsheet.Sheets {
		sheetName := sheet.Properties.Title
		log.Printf("Sheet Name: %s\n", sheetName)

		readRange := fmt.Sprintf("%s!B:E", sheetName)
		resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
		if err != nil {
			log.Printf("Unable to read data from sheet '%s': %v", sheetName, err)
			continue
		}

		for _, row := range resp.Values {
			if len(row) < 5 {
				continue
			}

			dateStr, ok := row[0].(string)
			if !ok {
				continue
			}

			dateParsed, err := time.Parse("02/01/2006", dateStr)
			if err != nil {
				log.Printf("Skipping invalid date: %v", err)
				continue
			}

			if dateParsed.Equal(dateToFind) || dateParsed.After(dateToFind) {
				// Weight is in column E (index 4)
				weight, ok := row[3].(string)
				if ok {
					fmt.Fprintf(w, "Weight on %v in sheet '%s': %s\n", dateToFind, sheetName, weight)
					return
				}
			}
		}
	}
}
