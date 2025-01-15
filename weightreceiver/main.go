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

var stabilizationDuration = 3 * time.Second

func onlyWeightData(rawData []byte) float64 {
	if len(rawData) < 19 {
		return -1
	}
	weightBytes := rawData[17:19]
	return float64(int(weightBytes[1])<<8|int(weightBytes[0])) / 100.0
}

func main() {
	adapter := bluetooth.DefaultAdapter
	if adapter.Enable() != nil {
		fmt.Println("Failed to enable Bluetooth adapter")
		return
	}

	signalChan := make(chan os.Signal, 1)
	weightDetected := make(chan float64, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		adapter.StopScan()
		fmt.Println("Scan interrupted, exiting...")
		os.Exit(0)
	}()

	var currentWeight float64 = -1
	var lastStableTime time.Time
	isStable := false

	go func() {
		log.Println("Scanning...")
		adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			address := strings.ToUpper(result.Address.String())
			if address == targetMACAddress {
				rawWeight := onlyWeightData(result.ManufacturerData()[0].Data)

				if currentWeight != -1 && rawWeight == currentWeight {
					log.Printf("Weight has been stabled on %.2f for %.0f seconds", rawWeight, time.Since(lastStableTime).Seconds())
					if time.Since(lastStableTime) >= stabilizationDuration && !isStable {
						isStable = true
						weightDetected <- rawWeight
					}
				} else {
					currentWeight = rawWeight
					lastStableTime = time.Now()
					isStable = false
				}
			}
		})
	}()

	select {
	case stableWeight := <-weightDetected:
		adapter.StopScan()
		fmt.Printf("Stable weight detected: %.2fKg\n", stableWeight)
	}
}
