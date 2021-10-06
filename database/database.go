package database

import (
	"fmt"
	"log"
	"lumaghg/dualis-crawler/crawler"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type App struct {
	dynamoClient *dynamodb.DynamoDB
	tableName    string
}

type DynamoCourse struct {
	Email      string
	Entry_type string
	Courses    []crawler.Course
}

func CompareGrades(courses []crawler.Course, email string) {
	//create Session and Client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(sess)

	gradesApp := App{dynamoClient: svc, tableName: "DUALIS_GRADES"}

	gradesApp.updateDatabase(courses, email)
}

func (app *App) updateDatabase(courses []crawler.Course, email string) ([]crawler.Course, error) {

	client := app.dynamoClient
	result, err := client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(app.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Email": {
				S: aws.String(email),
			},
		},
	})
	if err != nil {
		log.Fatalf("Got error calling GetItem: %s", err)
	}

	oldCourses := []crawler.Course{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &oldCourses)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	fmt.Println(oldCourses)

	//wrap courses with dynamo keys
	dynamoCourses := DynamoCourse{Email: email, Courses: courses}
	//transform into dynamo compatible type
	coursesMap, err := dynamodbattribute.MarshalMap(dynamoCourses)
	if err != nil {
		log.Fatalf("Got error marshalling map: %s", err)
	}
	//insert item into database
	input := &dynamodb.PutItemInput{
		Item:      coursesMap,
		TableName: aws.String(app.tableName),
	}

	_, err = client.PutItem(input)
	if err != nil {
		log.Fatalf("Got error calling PutItem: %s", err)
		return []crawler.Course{}, err
	}

	return []crawler.Course{}, nil
}
