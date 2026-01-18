package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

type WeightScanner struct {
	btAdapter     *bluetooth.Adapter
	weightUpdater IWeightUpdater
}

const spreadsheetId = "1WBAPDBnb01eJFBpBwmo_NDq2r9B44U0wyt7CZFxtus4"
const targetMACAddress = "ED:67:39:0A:C5:C0"
const stabilizationDuration = 2 * time.Second
const dryWindowDuration = 10 * time.Second

func NewWeightScanner() (*WeightScanner, error) {
	return &WeightScanner{
		btAdapter:     bluetooth.DefaultAdapter,
		weightUpdater: NewGSWeightUpdater(),
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
				log.Printf("Found a stable weight: %.2f Kg, sending it for storage ", scannedWeight)
				finalWeightDetected <- scannedWeight
				lastStableTime = time.Now()
				currentStableWeight = -1
				dryWindowStart = time.Now()
			}
		} else {
			currentStableWeight = scannedWeight
			lastStableTime = time.Now()
		}
		log.Printf("Done with processing weight %.2f\n", scannedWeight)
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

func main() {
	osSignals := make(chan os.Signal, 1)
	finalWeightDetected := make(chan float32, 5)
	incomingWeights := make(chan float32, 5)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	ws, err := NewWeightScanner()
	if err != nil {
		log.Fatal(err)
	}

	go ws.interruptOnOsSignals(osSignals)
	go ws.scanWeights(incomingWeights)
	go processWeights(incomingWeights, finalWeightDetected)

	for finalWeight := range finalWeightDetected {
		ws.weightUpdater.Update(spreadsheetId, time.Now(), finalWeight)
		fmt.Printf("Stable weight detected: %.2fKg\n", finalWeight)
	}
	fmt.Println("Channel closed, stopping weight detection.")
}
