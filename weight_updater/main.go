package main

import (
	"andrea-aus-hh.de/weightlambda/weight_updater_function"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", weight_updater_function.WeightWriter)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
