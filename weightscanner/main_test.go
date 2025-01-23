package main

import (
	"sync"
	"testing"
	"time"
)

func TestProducerConsumer(t *testing.T) {
	var wg sync.WaitGroup
	sharedResource := 0

	runTest := func(t *testing.T, incomingWeights <-chan float32, finalWeightDetected chan float32, weightProvider func(), expectedSuccess bool, expectedWeight float32) {
		go processWeights(incomingWeights, finalWeightDetected)
		wg.Add(1)
		go func() {
			defer wg.Done()
		}()
		go weightProvider()
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
		incomingWeights := make(chan float32, 1)
		finalWeightDetected := make(chan float32, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float32{1., 2., 3., 4., 5.} {
				incomingWeights <- data
			}
			close(incomingWeights)
		}, false, 0.)
	})

	t.Run(`When passing five same weights without enough time,
					then no result is returned`, func(t *testing.T) {
		incomingWeights := make(chan float32, 1)
		finalWeightDetected := make(chan float32, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float32{10., 10., 10., 10., 10., 10., 10.} {
				incomingWeights <- data
			}
			close(incomingWeights)
		}, false, 0.)
	})

	t.Run(`When passing a different weight every two seconds,
					then no result is returned`, func(t *testing.T) {
		incomingWeights := make(chan float32, 1)
		finalWeightDetected := make(chan float32, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float32{10., 20., 30., 40.} {
				incomingWeights <- data
				time.Sleep(2 * time.Second)
			}
			close(incomingWeights)
		}, false, 0.)
	})

	t.Run(`When passing a different weight every two seconds,
						and then the same weight every two seconds for two time
					then that weight is returned`, func(t *testing.T) {
		incomingWeights := make(chan float32, 1)
		finalWeightDetected := make(chan float32, 1)
		runTest(t, incomingWeights, finalWeightDetected, func() {
			for _, data := range []float32{10., 20., 80., 80., 80.} {
				incomingWeights <- data
				time.Sleep(2 * time.Second)
			}
			close(incomingWeights)
		}, true, 80.)
	})

	t.Run(`When passing a different weight every two seconds,
						and then 30 Kg every two seconds for three times
						and then 50 Kg every two seconds for four times
						and then 10 Kg every two seconds for four times
					then 30 and 10 Kg are returned, because 50 Kg is in the dry window`, func(t *testing.T) {

		incomingWeights := make(chan float32, 5)
		finalWeightDetected := make(chan float32, 5)
		go processWeights(incomingWeights, finalWeightDetected)
		wg.Add(1)
		go func() {
			defer wg.Done()
		}()
		go func() {
			for _, data := range []float32{10., 20., 30., 30., 30., 50., 50., 50., 50., 10., 10., 10., 10.} {
				incomingWeights <- data
				time.Sleep(2 * time.Second)
			}
			close(incomingWeights)
		}()
		wg.Wait()
		t.Logf("Shared resource value: %d", sharedResource)

		// Then
		result, _ := <-finalWeightDetected
		if result != 30. {
			t.Errorf("Wrong result: %.2f, expected 30.", result)
		}

		result, _ = <-finalWeightDetected
		if result != 10. {
			t.Errorf("Wrong result: %.2f, expected 10. ", result)
		}

	})
}
