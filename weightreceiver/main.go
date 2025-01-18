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

const targetMACAddress = "ED:67:39:0A:C5:C0"

var adapter = bluetooth.DefaultAdapter
var stabilizationDuration = 3 * time.Second

func parseWeightData(rawData []byte) float64 {
	if len(rawData) < 19 {
		return -1
	}
	weightBytes := rawData[17:19]
	return float64(int(weightBytes[1])<<8|int(weightBytes[0])) / 100.0
}

func processWeights(incomingWeights <-chan float64, finalWeightDetected chan<- float64) {
	var currentWeight float64 = -1
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

func scanWeights(incomingWeights chan<- float64) {
	if adapter.Enable() != nil {
		fmt.Println("Failed to enable Bluetooth adapter")
		close(incomingWeights)
		return
	}
	log.Println("Scanning...")
	adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		address := strings.ToUpper(result.Address.String())
		if address == targetMACAddress {
			incomingWeights <- parseWeightData(result.ManufacturerData()[0].Data)
		}
	})
	close(incomingWeights)
	log.Println("Scan stopped!")
}

func interruptOnOsSignals(osSignals <-chan os.Signal) {
	<-osSignals
	adapter.StopScan()
	fmt.Println("Scan interrupted, exiting...")
	os.Exit(0)
}

func main() {
	osSignals := make(chan os.Signal, 1)
	finalWeightDetected := make(chan float64, 1)
	incomingWeights := make(chan float64, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	go interruptOnOsSignals(osSignals)
	go scanWeights(incomingWeights)
	go processWeights(incomingWeights, finalWeightDetected)

	select {
	case finalWeight, ok := <-finalWeightDetected:
		adapter.StopScan()
		if ok {
			fmt.Printf("Stable weight detected: %.2fKg\n", finalWeight)
		} else {
			fmt.Printf("Stable weight not detected")
		}
		os.Exit(0)
	}
}
