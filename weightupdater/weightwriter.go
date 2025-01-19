package weightupdater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/sheets/v4"
)

var handler *WeightWriterHandler

type WeightWriterHandler struct {
	WeightService WeightService
}

func init() {
	ctx := context.Background()
	sheetsService, err := sheets.NewService(ctx)
	if err != nil {
		log.Fatalf("Unable to create Sheets client: %v", err)
	}

	handler = &WeightWriterHandler{
		WeightService: &GoogleSheetService{service: sheetsService},
	}
	functions.HTTP("WeightWriter", WeightWriter)
}

func WeightWriter(w http.ResponseWriter, r *http.Request) {
	handler.WeightWriter(w, r)
}

type RequestBody struct {
	Date   time.Time `json:"date"`
	Weight float32   `json:"weight"`
}

func (r *RequestBody) UnmarshalJSON(data []byte) error {
	type Alias RequestBody
	aux := &struct {
		Date string `json:"date"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	parsedDate, err := time.Parse("2006-01-02", aux.Date)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}
	r.Date = parsedDate
	r.Weight = aux.Weight
	return nil
}

func (rb RequestBody) validate() error {
	if rb.Weight < 60 {
		return errors.New("weight too low")
	}
	if rb.Weight > 120 {
		return errors.New("weight too high")
	}
	if rb.Date.Before(time.Date(2021, 3, 21, 0, 0, 0, 0, time.UTC)) {
		return errors.New("date is too much in the past")
	}
	return nil
}

// WeightWriter is an HTTP Cloud Function with a request parameter.
func (h *WeightWriterHandler) WeightWriter(w http.ResponseWriter, r *http.Request) {
	spreadsheetId := "1WBAPDBnb01eJFBpBwmo_NDq2r9B44U0wyt7CZFxtus4"

	var body RequestBody
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&body)
	if err != nil {
		http.Error(w, "Failed to parse JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err = body.validate()
	if err != nil {
		http.Error(w, "Failed to validate JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("body: %v", body)

	err = h.WeightService.Update(spreadsheetId, body.Date, body.Weight)
	if err != nil {
		fmt.Fprintf(w, "Unable to update Weight: %v", err)
	}
}

func datesAreEqual(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.Month() == date2.Month() && date1.Day() == date2.Day()
}
