package main

import (
	"fmt"
	"lumaghg/dualis-crawler/crawler"
	"lumaghg/dualis-crawler/database"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type MyEvent struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	NotificationEmail string `json:"notificationEmail"`
}

func HandleRequest(event MyEvent) {
	start := time.Now()

	results, _ := crawler.GetDualisCrawlResults(event.Email, event.Password)
	fmt.Println(time.Since(start))
	dualisChanges, err := database.CompareAndUpdateCourses(results, event.Email)
	if err != nil {
		fmt.Println(err)
	}
	err = database.SendUpdateEmail(dualisChanges, event.NotificationEmail)
	return
}

func main() {
	lambda.Start(HandleRequest)
}
