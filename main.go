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

func HandleRequest(event MyEvent) error {
	start := time.Now()

	results, _ := crawler.GetDualisCrawlResults(event.Email, event.Password)
	fmt.Println(time.Since(start))
	dualisChanges, err := database.CompareAndUpdateCourses(results, event.Email)
	fmt.Println(dualisChanges)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if len(dualisChanges) > 1 {
		err = database.SendUpdateEmail(dualisChanges, event.NotificationEmail)
	}
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
