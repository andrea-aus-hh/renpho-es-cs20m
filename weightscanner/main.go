package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/api/idtoken"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

type WeightScanner struct {
	httpClient       *http.Client
	weightUpdaterUrl string
	btAdapter        *bluetooth.Adapter
}

const targetMACAddress = "ED:67:39:0A:C5:C0"

const stabilizationDuration = 2 * time.Second

func NewWeightScanner() (*WeightScanner, error) {
	url := os.Getenv("WEIGHTUPDATER_URL")
	if url == "" {
		return nil, fmt.Errorf("WEIGHTUPDATER_URL not set")
	}

	client, err := idtoken.NewClient(context.Background(), url)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise the client: %w", err)
	}

	return &WeightScanner{
		httpClient:       client,
		weightUpdaterUrl: url,
		btAdapter:        bluetooth.DefaultAdapter,
	}, nil
}

func parseWeightData(rawData []byte) float32 {
	if len(rawData) < 19 {
		return -1
	}
	weightBytes := rawData[17:19]
	return float32(int(weightBytes[1])<<8|int(weightBytes[0])) / 100.0
}

func processWeights(incomingWeights <-chan float32, finalWeightDetected chan<- float32) {
	var currentWeight float32 = -1
	var lastStableTime time.Time
	for rawWeight := range incomingWeights {
		log.Printf("Received weight %.2f", rawWeight)
		if currentWeight != -1 && rawWeight == currentWeight {
			log.Printf("Weight has been stable on %.2f for %.0f seconds", rawWeight, time.Since(lastStableTime).Seconds())
			if time.Since(lastStableTime) >= stabilizationDuration {
				finalWeightDetected <- rawWeight
			}
		} else {
			currentWeight = rawWeight
			lastStableTime = time.Now()
		}
	}
	log.Println("Channel closed.")
	close(finalWeightDetected)
}

func (ws *WeightScanner) scanWeights(incomingWeights chan<- float32) {
	if ws.btAdapter.Enable() != nil {
		fmt.Println("Failed to enable Bluetooth btAdapter")
		close(incomingWeights)
		return
	}
	log.Println("Scanning...")
	ws.btAdapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		address := strings.ToUpper(result.Address.String())
		if address == targetMACAddress {
			incomingWeights <- parseWeightData(result.ManufacturerData()[0].Data)
		}
	})
	close(incomingWeights)
	log.Println("Scan stopped!")
}

func (ws *WeightScanner) interruptOnOsSignals(osSignals <-chan os.Signal) {
	<-osSignals
	ws.btAdapter.StopScan()
	fmt.Println("Scan interrupted, exiting...")
	os.Exit(0)
}

type RequestBody struct {
	Date   time.Time `json:"date"`
	Weight float32   `json:"weight"`
}

func (r RequestBody) MarshalJSON() ([]byte, error) {
	type Alias RequestBody
	return json.Marshal(&struct {
		Date string `json:"date"`
		*Alias
	}{
		Date:  r.Date.Format("2006-01-02"),
		Alias: (*Alias)(&r),
	})
}

func (ws *WeightScanner) sendWeight(detectedWeight float32) {
	log.Printf("Sending weight %.2f to %s", detectedWeight, ws.weightUpdaterUrl)
	body := RequestBody{Weight: detectedWeight, Date: time.Now()}
	jsonData, err := json.Marshal(body)
	log.Printf("JSON: %s", string(jsonData))

	req, err := http.NewRequest("POST", ws.weightUpdaterUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		panic(err)
	}

	resp, err := ws.httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response: %s\n", resp.Status)
}

func main() {
	osSignals := make(chan os.Signal, 1)
	finalWeightDetected := make(chan float32, 1)
	incomingWeights := make(chan float32, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	wr, err := NewWeightScanner()
	if err != nil {
		log.Fatal(err)
	}

	go wr.interruptOnOsSignals(osSignals)
	go wr.scanWeights(incomingWeights)
	go processWeights(incomingWeights, finalWeightDetected)

	select {
	case finalWeight, ok := <-finalWeightDetected:
		wr.sendWeight(finalWeight)
		wr.btAdapter.StopScan()
		if ok {
			fmt.Printf("Stable weight detected: %.2fKg\n", finalWeight)
		} else {
			fmt.Printf("Stable weight not detected")
		}
		os.Exit(0)
	}
}
