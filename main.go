package main

import (
	"encoding/json"
	"fmt"
	"lumaghg/dualis-crawler/crawler"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type MyEvent struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	NotificationEmail string `json:"notificationEmail"`
}

func HandleRequest(event MyEvent) (string, error) {
	start := time.Now()

	results, _ := crawler.GetDualisCrawlResults(event.Email, event.Password)
	fmt.Println(time.Since(start))
	fmt.Println(results)
	resultsBytes, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(resultsBytes), nil

	/*
		dualisChanges, err := database.UpdateDatabaseAndGetChanges(results, event.Email)
		fmt.Println(dualisChanges)
		if err != nil {
			fmt.Println(err)
			return err
		}
		if len(dualisChanges) > 0 {
			err = email.SendUpdateEmail(dualisChanges, event.NotificationEmail)
		}
		if err != nil {
			fmt.Println(err)
			return err
		}
		return nil
	*/
}

func main() {
	//HandleRequest(MyEvent{Email: "s201808@student.dhbw-mannheim.de", Password: "xj3ghgPUx"})
	lambda.Start(HandleRequest)
}
