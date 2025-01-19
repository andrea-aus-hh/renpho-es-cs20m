package main

import (
	"andrea-aus-hh.de/weightupdater"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", weightupdater.WeightUpdater)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
