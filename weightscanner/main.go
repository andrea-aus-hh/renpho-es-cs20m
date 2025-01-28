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
const dryWindowDuration = 10 * time.Second

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
	var currentStableWeight float32 = -1
	var lastStableTime time.Time
	var dryWindowStart time.Time
	for scannedWeight := range incomingWeights {
		if time.Since(dryWindowStart) < dryWindowDuration {
			log.Printf("Skipping weight %.2f because of dry-window", scannedWeight)
			continue
		}
		log.Printf("Scanned weight %.2f", scannedWeight)
		if currentStableWeight != -1 && scannedWeight == currentStableWeight {
			log.Printf("Weight has been stable on %.2f Kg for %.0f seconds", scannedWeight, time.Since(lastStableTime).Seconds())
			if time.Since(lastStableTime) >= stabilizationDuration {
				log.Printf("Found a stable weight: %.2f Kg. will pass it ", scannedWeight)
				finalWeightDetected <- scannedWeight
				lastStableTime = time.Now()
				currentStableWeight = -1
				dryWindowStart = time.Now()
			}
		} else {
			currentStableWeight = scannedWeight
			lastStableTime = time.Now()
		}
		log.Printf("Done with processing weight %.2f", scannedWeight)
	}
	log.Println("Scanner channel closed, will stop processing.")
	close(finalWeightDetected)
}

func (ws *WeightScanner) scanWeights(incomingWeights chan<- float32) {
	if ws.btAdapter.Enable() != nil {
		fmt.Println("Failed to enable Bluetooth Adapter")
		close(incomingWeights)
		return
	}
	log.Println("Scanning...")
	err := ws.btAdapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		address := strings.ToUpper(result.Address.String())
		if address == targetMACAddress {
			incomingWeights <- parseWeightData(result.ManufacturerData()[0].Data)
		}
	})
	if err != nil {
		close(incomingWeights)
		log.Fatal(err)
	}
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
	finalWeightDetected := make(chan float32, 5)
	incomingWeights := make(chan float32, 5)
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
		if ok {
			fmt.Printf("Stable weight detected: %.2fKg\n", finalWeight)
		} else {
			fmt.Printf("Stable weight not detected")
		}
		os.Exit(0)
	}
}
