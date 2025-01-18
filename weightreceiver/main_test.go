package main

import (
	"sync"
	"testing"
	"time"
)

func TestProducerConsumer(t *testing.T) {
	var wg sync.WaitGroup
	sharedResource := 0

	runTest := func(t *testing.T, incomingWeights <-chan float64, finalWeightDetected chan float64, weightProvider func(), expectedSuccess bool, expectedWeight float64) {
		go processWeights(incomingWeights, finalWeightDetected)
		wg.Add(1)
		go func() {
			defer wg.Done()
			weightProvider()
		}()
		wg.Wait()
		t.Logf("Shared resource value: %d", sharedResource)
		result, ok := <-finalWeightDetected
		if expectedSuccess != ok {
			t.Errorf("Expected: %t, got: %t", expectedSuccess, ok)
		}
		if result != expectedWeight {
			t.Errorf("Wrong result: %.2f", result)
		}
	}

	t.Run(`When passing five different weights without enough time,
					then no result is returned`, func(t *testing.T) {
		incomingWeights := make(chan float64, 1)
		finalWeightDetected := make(chan float64, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float64{1., 2., 3., 4., 5.} {
				incomingWeights <- data
			}
			close(incomingWeights)
		}, false, 0.)
	})

	t.Run(`When passing five same weights without enough time,
					then no result is returned`, func(t *testing.T) {
		incomingWeights := make(chan float64, 1)
		finalWeightDetected := make(chan float64, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float64{10., 10., 10., 10., 10., 10., 10.} {
				incomingWeights <- data
			}
			close(incomingWeights)
		}, false, 0.)
	})

	t.Run(`When passing a different weight every two seconds,
					then no result is returned`, func(t *testing.T) {
		incomingWeights := make(chan float64, 1)
		finalWeightDetected := make(chan float64, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float64{10., 20., 30., 40.} {
				incomingWeights <- data
				time.Sleep(2 * time.Second)
			}
			close(incomingWeights)
		}, false, 0.)
	})

	t.Run(`When passing a different weight every two seconds,
						and then the same weight every two seconds for two time
					then that weight is returned`, func(t *testing.T) {
		incomingWeights := make(chan float64, 1)
		finalWeightDetected := make(chan float64, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float64{10., 20., 30., 40., 50., 50., 50.} {
				incomingWeights <- data
				time.Sleep(2 * time.Second)
			}
			close(incomingWeights)
		}, true, 50.)
	})
}
