package database

import (
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
	email      string
	entry_type string
	courses    []crawler.Course
}

func CheckNewGrades(courses []crawler.Course, email string) {
	//create Session and Client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(sess)

	app := App{dynamoClient: svc, tableName: "DUALIS_STORE"}

	app.updateDatabase(courses, email)
}

func (app *App) updateDatabase(courses []crawler.Course, email string) ([]crawler.Course, error) {
	//wrap courses with dynamo keys
	client := app.dynamoClient
	dynamoCourses := DynamoCourse{email: email, entry_type: "GRADE", courses: courses}
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
