package main

import (
	"encoding/json"
	"fmt"
	"lumaghg/dualis-crawler/crawler"
	"lumaghg/dualis-crawler/database"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
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
	err = database.CompareAndUpdateCourses(results, event.Email)
	if err != nil {
		return MyResponse{}, nil
	}

	return MyResponse{body: string(jsonResults)}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
