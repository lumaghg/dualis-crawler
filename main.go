package main

import (
	"encoding/json"
	"fmt"
	"lumaghg/dualis-crawler/crawler"
	"time"
)

type MyEvent struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type MyResponse struct {
	body string
}

func HandleRequest(event MyEvent) (MyResponse, error) {
	start := time.Now()

	results, _ := crawler.GetDualisCrawlResults(event.Email, event.Password)
	jsonResults, err := json.Marshal(results)
	fmt.Println(time.Since(start))
	if err != nil {
		return MyResponse{}, err
	}
	return MyResponse{body: string(jsonResults)}, nil
}

func main() {
	//lambda.Start(HandleRequest)
	HandleRequest(MyEvent{Email: "s201808@student.dhbw-mannheim.de", Password: "xj3ghgPUx"})
}
