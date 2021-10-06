package database

import (
	"fmt"
	"log"
	"lumaghg/dualis-crawler/crawler"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type App struct {
	dynamoClient *dynamodb.DynamoDB
	tableName    string
	email        string
}

type DynamoCourse struct {
	Email      string
	Entry_type string
	Courses    []crawler.Course
}

func CompareAndUpdateCourses(courses []crawler.Course, email string) error {
	//create Session and Client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(sess)

	gradesApp := App{dynamoClient: svc, tableName: "DUALIS_GRADES", email: email}

	dualisChanges, err := gradesApp.updateDatabaseAndGetChanges(courses)
	if err != nil {
		return err
	}
	err = gradesApp.SendUpdateEmail(dualisChanges)
	return err
}

func (app *App) updateDatabaseAndGetChanges(newCourses []crawler.Course) ([]crawler.Course, error) {

	client := app.dynamoClient
	//get the item from the database and convert it to course struct
	result, err := client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(app.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Email": {
				S: aws.String(app.email),
			},
		},
	})
	if err != nil {
		log.Fatalf("Got error calling GetItem: %s", err)
		return []crawler.Course{}, nil
	}

	dynamoItem := DynamoCourse{}

	err = dynamodbattribute.UnmarshalMap(result.Item, &dynamoItem)
	if err != nil {
		panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
	}

	oldCourses := dynamoItem.Courses
	fmt.Println(oldCourses)
	differentCourses := []crawler.Course{}
	/**
	* If the retrieved Item is empty, leave the differentCourses empty
	* because that means this is the first request for the user and sending
	* an Email for all grades on the first Call is useless (not real changes)
	 */
	if len(oldCourses) > 0 {
		differentCourses = getCourseDifferences(oldCourses, newCourses)
	} else {

		//wrap courses with dynamo keys
		dynamoCourses := DynamoCourse{Email: app.email, Courses: newCourses}
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

	}
	return differentCourses, nil
}

func getCourseDifferences(oldCourses []crawler.Course, newCourses []crawler.Course) []crawler.Course {
	courseDifferences := []crawler.Course{}
	for _, newCourse := range newCourses {
		containsCourse := false
		for _, oldCourse := range oldCourses {
			if reflect.DeepEqual(newCourse, oldCourse) {
				containsCourse = true
			}
		}

		if !containsCourse {
			courseDifferences = append(courseDifferences, newCourse)
		}
	}
	return courseDifferences
}

func (app *App) SendUpdateEmail(dualisChanges []crawler.Course) error {
	fmt.Println("send email")
	return nil
}
