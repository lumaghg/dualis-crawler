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

const (
	CourseTableName = "DUALIS_GRADES"
)

type DynamoCourse struct {
	Email   string
	Courses []crawler.Course
}

func UpdateDatabaseAndGetChanges(newCourses []crawler.Course, email string) ([]crawler.Course, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := dynamodb.New(sess)

	//get the item from the database and convert it to course struct
	result, err := client.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(CourseTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"Email": {
				S: aws.String(email),
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
	differentCourses := []crawler.Course{}
	/**
	* If the retrieved Item is empty, leave the differentCourses empty
	* because that means this is the first request for the user and sending
	* an Email for all grades on the first Call is useless (not real changes)
	 */
	if len(oldCourses) > 0 {
		differentCourses = getCourseDifferences(oldCourses, newCourses)

		//wrap courses with dynamo keys
		dynamoCourses := DynamoCourse{Email: email, Courses: newCourses}
		//transform into dynamo compatible type
		coursesMap, err := dynamodbattribute.MarshalMap(dynamoCourses)
		if err != nil {
			log.Fatalf("Got error marshalling map: %s", err)
		}
		//if an item exists, replace it with the new, if not, create it
		input := &dynamodb.PutItemInput{
			Item:      coursesMap,
			TableName: aws.String(CourseTableName),
		}

		_, err = client.PutItem(input)
		if err != nil {
			log.Fatalf("Got error calling PutItem: %s", err)
			return []crawler.Course{}, err
		}

	}
	fmt.Println("dualis changes: \n", differentCourses)
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
