package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type IWeightUpdater interface {
	Update(spreadsheetId string, date time.Time, weight float32) error
}

type GSWeightUpdater struct {
	credsFile string
}

type serviceAccountCreds struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

func NewGSWeightUpdater() *GSWeightUpdater {
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
	}
	return &GSWeightUpdater{
		credsFile: credsFile,
	}
}

const dateLayout = "02/01/2006"
const weightDiaryPrefix = "Diario"
const firstDatePosition = "B4"
const datesRange = "B:B"
const weightColumn = "E"
const sheetsScope = "https://www.googleapis.com/auth/spreadsheets"

func (g *GSWeightUpdater) getAccessToken() (string, error) {
	credsData, err := os.ReadFile(g.credsFile)
	if err != nil {
		return "", fmt.Errorf("reading creds file: %w", err)
	}

	var creds serviceAccountCreds
	if err := json.Unmarshal(credsData, &creds); err != nil {
		return "", fmt.Errorf("parsing creds: %w", err)
	}

	now := time.Now()
	jwtHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]interface{}{
		"iss":   creds.ClientEmail,
		"scope": sheetsScope,
		"aud":   creds.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})
	jwtClaims := base64.RawURLEncoding.EncodeToString(claims)
	signingInput := jwtHeader + "." + jwtClaims

	block, _ := pem.Decode([]byte(creds.PrivateKey))
	if block == nil {
		return "", errors.New("failed to decode PEM block from private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("private key is not RSA")
	}

	hash := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(nil, rsaKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("signing JWT: %w", err)
	}
	jwt := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	resp, err := http.PostForm(creds.TokenURI, url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {jwt},
	})
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed (%d): %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// sheetsAPIRequest makes an authenticated request to the Sheets API.
func (g *GSWeightUpdater) sheetsAPIRequest(method, apiURL string, body io.Reader) ([]byte, error) {
	token, err := g.getAccessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, apiURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Sheets API error (%d): %s", resp.StatusCode, respBody)
	}
	return respBody, nil
}

type spreadsheetMeta struct {
	Sheets []sheetMeta `json:"sheets"`
}

type sheetMeta struct {
	Properties struct {
		Title string `json:"title"`
	} `json:"properties"`
}

type valuesResponse struct {
	Values [][]interface{} `json:"values"`
}

func (g *GSWeightUpdater) getSpreadsheetMeta(spreadsheetId string) (*spreadsheetMeta, error) {
	apiURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s?fields=sheets.properties.title", spreadsheetId)
	body, err := g.sheetsAPIRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	var meta spreadsheetMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (g *GSWeightUpdater) getValues(spreadsheetId, readRange string) (*valuesResponse, error) {
	apiURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s",
		spreadsheetId, url.PathEscape(readRange))
	body, err := g.sheetsAPIRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	var vals valuesResponse
	if err := json.Unmarshal(body, &vals); err != nil {
		return nil, err
	}
	return &vals, nil
}

func (g *GSWeightUpdater) putValues(spreadsheetId, writeRange string, values [][]interface{}) error {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"values": values,
	})
	apiURL := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s/values/%s?valueInputOption=USER_ENTERED",
		spreadsheetId, url.PathEscape(writeRange))
	_, err := g.sheetsAPIRequest("PUT", apiURL, strings.NewReader(string(reqBody)))
	return err
}

func (g *GSWeightUpdater) Update(spreadsheetId string, date time.Time, weight float32) error {
	meta, err := g.getSpreadsheetMeta(spreadsheetId)
	if err != nil {
		log.Printf("Couldn't connect to the spreadsheet: %v\n", err)
		return err
	}

	writeRange, err := g.findWriteRangeForDate(spreadsheetId, meta, date)
	if err != nil {
		return err
	}

	err = g.putValues(spreadsheetId, writeRange, [][]interface{}{{formatFloat(weight)}})
	if err != nil {
		log.Printf("Unable to update at range '%s': %v", writeRange, err)
		return err
	}
	log.Printf("Updated sheet '%s' at location '%s'\n", writeRange, writeRange)
	return nil
}

func datesAreEqual(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.Month() == date2.Month() && date1.Day() == date2.Day()
}

func (g *GSWeightUpdater) findWriteRangeForDate(spreadsheetId string, meta *spreadsheetMeta, date time.Time) (string, error) {
	candidateSheet := g.findCorrectSheet(spreadsheetId, meta, date)

	log.Printf("Candidate sheet is called '%s'", candidateSheet.Properties.Title)

	readRange := fmt.Sprintf("%s!%s", candidateSheet.Properties.Title, datesRange)
	resp, err := g.getValues(spreadsheetId, readRange)
	if err != nil {
		log.Printf("Unable to read data from sheet '%s': %v", candidateSheet.Properties.Title, err)
		return "", err
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
			return "'" + candidateSheet.Properties.Title + "'" + "!" + weightColumn + strconv.Itoa(i+1), nil
		}
	}
	return "", errors.New("Couldn't find date " + date.String())
}

func (g *GSWeightUpdater) findCorrectSheet(spreadsheetId string, meta *spreadsheetMeta, dateToFind time.Time) sheetMeta {
	candidateSheet := meta.Sheets[0]
	for _, sheet := range meta.Sheets {
		if !strings.HasPrefix(sheet.Properties.Title, weightDiaryPrefix) {
			break
		}
		readRange := fmt.Sprintf("%s!%s", candidateSheet.Properties.Title, firstDatePosition)
		resp, err := g.getValues(spreadsheetId, readRange)
		if err != nil {
			log.Printf("Unable to retrieve data from sheet: %v", err)
			continue
		}
		if len(resp.Values) == 0 || len(resp.Values[0]) == 0 {
			continue
		}
		firstDateStr, ok := resp.Values[0][0].(string)
		if !ok {
			continue
		}
		firstDate, err := time.Parse(dateLayout, firstDateStr)
		if err != nil {
			log.Printf("Unable to parse data from sheet: %v", err)
			continue
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
