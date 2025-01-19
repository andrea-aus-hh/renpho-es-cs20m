package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

type WeightReceiver struct {
	httpClient       *http.Client
	weightUpdaterUrl string
	btAdapter        *bluetooth.Adapter
}

const targetMACAddress = "ED:67:39:0A:C5:C0"

var stabilizationDuration = 3 * time.Second

func NewWeightReceiver() (*WeightReceiver, error) {
	keyFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if keyFile == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	creds, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}

	config, err := google.JWTConfigFromJSON(creds, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, err
	}

	url := os.Getenv("WEIGHTUPDATER_URL")
	if url == "" {
		return nil, fmt.Errorf("WEIGHTUPDATER_URL not set")
	}

	return &WeightReceiver{
		httpClient:       oauth2.NewClient(context.Background(), config.TokenSource(context.Background())),
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
	isStable := false
	for rawWeight := range incomingWeights {
		log.Printf("Received weight %.2f", rawWeight)
		if currentWeight != -1 && rawWeight == currentWeight {
			log.Printf("Weight has been stabled on %.2f for %.0f seconds", rawWeight, time.Since(lastStableTime).Seconds())
			if time.Since(lastStableTime) >= stabilizationDuration && !isStable {
				isStable = true
				finalWeightDetected <- rawWeight
			}
		} else {
			currentWeight = rawWeight
			lastStableTime = time.Now()
			isStable = false
		}
	}
	log.Println("Channel closed.")
	close(finalWeightDetected)
}

func (wr *WeightReceiver) scanWeights(incomingWeights chan<- float32) {
	if wr.btAdapter.Enable() != nil {
		fmt.Println("Failed to enable Bluetooth btAdapter")
		close(incomingWeights)
		return
	}
	log.Println("Scanning...")
	wr.btAdapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		address := strings.ToUpper(result.Address.String())
		if address == targetMACAddress {
			incomingWeights <- parseWeightData(result.ManufacturerData()[0].Data)
		}
	})
	close(incomingWeights)
	log.Println("Scan stopped!")
}

func (wr *WeightReceiver) interruptOnOsSignals(osSignals <-chan os.Signal) {
	<-osSignals
	wr.btAdapter.StopScan()
	fmt.Println("Scan interrupted, exiting...")
	os.Exit(0)
}

type RequestBody struct {
	Date   time.Time `json:"date"`
	Weight float32   `json:"weight"`
}

func (wr *WeightReceiver) sendWeight(detectedWeight float32) {
	body := RequestBody{Weight: detectedWeight, Date: time.Now()}
	jsonData, err := json.Marshal(body)

	req, err := http.NewRequest("GET", wr.weightUpdaterUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		panic(err)
	}

	resp, err := wr.httpClient.Do(req)
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

	wr, err := NewWeightReceiver()
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
