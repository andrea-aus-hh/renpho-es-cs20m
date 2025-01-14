package main

import (
	"errors"
	"fmt"
	"google.golang.org/api/sheets/v4"
	"log"
	"strings"
	"time"
)

type WeightService interface {
	Update(spreadsheetId string, date time.Time, weight float32) error
}

type GoogleSheetService struct {
	service *sheets.Service
}

const dateLayout = "02/01/2006"
const weightDiaryPrefix = "Diario"
const firstDatePosition = "B4"
const datesRange = "B:B"
const weightColumn = "E"

func (g *GoogleSheetService) Update(spreadsheetId string, date time.Time, weight float32) error {
	spreadsheet, err := g.service.Spreadsheets.Get(spreadsheetId).Do()
	writeRange, err := g.findWriteRangeForDate(spreadsheet, date)

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{formatFloat(weight)}},
	}
	_, err = g.service.Spreadsheets.Values.Update(spreadsheetId, writeRange, valueRange).Do()
	if err != nil {
		log.Printf("Unable to update at range '%s': %v", writeRange, err)
		return err
	} else {
		log.Printf("Updated sheet '%s' at location '%s'\n", writeRange, writeRange)
	}
	return nil
}

func (g *GoogleSheetService) findWriteRangeForDate(spreadsheet *sheets.Spreadsheet, date time.Time) (string, error) {
	candidateSheet := g.findCorrectSheet(spreadsheet, date)

	log.Printf("Candidate sheet is called '%s'", candidateSheet.Properties.Title)

	readRange := fmt.Sprintf("%s!%s", candidateSheet.Properties.Title, datesRange)
	resp, err := g.service.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, readRange).Do()
	if err != nil {
		log.Printf("Unable to read data from sheet '%s': %v", candidateSheet.Properties.Title, err)
	}
	for i, row := range resp.Values {
		if len(row) == 0 {
			continue
		}
		dateStr, ok := row[0].(string)
		if !ok {
			continue
		}

		dateParsed, err := time.Parse(dateLayout, dateStr)
		if err != nil {
			continue
		}

		if datesAreEqual(dateParsed, date) {
			return candidateSheet.Properties.Title + "!" + weightColumn + string(rune(i+1)), nil
		}
	}
	return "", errors.New("Couldn't find date " + date.String())
}

func (g *GoogleSheetService) findCorrectSheet(spreadsheet *sheets.Spreadsheet, dateToFind time.Time) *sheets.Sheet {
	candidateSheet := spreadsheet.Sheets[0]
	for _, sheet := range spreadsheet.Sheets {
		if !strings.HasPrefix(sheet.Properties.Title, weightDiaryPrefix) {
			break
		}
		readRange := fmt.Sprintf("%s!%s", candidateSheet.Properties.Title, firstDatePosition)
		resp, err := g.service.Spreadsheets.Values.Get(spreadsheet.SpreadsheetId, readRange).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from sheet: %v", err)
		}
		firstDate, err := time.Parse(dateLayout, resp.Values[0][0].(string))
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

// formatFloat returns a float formatted with two decimal places, with a comma, like 23,45
func formatFloat(f float32) string {
	formatted := fmt.Sprintf("%.2f", f)
	return strings.Replace(formatted, ".", ",", 1)
}
