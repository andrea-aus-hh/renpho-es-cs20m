package sheet_updater_function

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/sheets/v4"
)

func init() {
	functions.HTTP("HelloHTTP", HelloHTTP)
}

// HelloHTTP is an HTTP Cloud Function with a request parameter.
func HelloHTTP(w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintf(w, "Spreadsheet Title: %s\n", spreadsheet.Properties.Title)
}
