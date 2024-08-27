package main

import (
	"AIChess/api"
	"net/http"
)

func main() {
	// Starts api on port 8081
	http.HandleFunc("/engine-move", api.StockFishEndpoint)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		return
	}
}
