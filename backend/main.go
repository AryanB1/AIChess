package main

import (
	"AIChess/api"
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("Hello, World!")
	http.HandleFunc("/engine-move", api.StockFishEndpoint)
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		return
	}
}
