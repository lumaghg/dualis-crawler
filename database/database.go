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
	/*
		compare the differences, so that new Courses and new Examinations and
		changes / updates in the exminations are found
		=> 1. find courses with the same name
		=> 2. check for every Examination of the new course, if it has a
			  deepEqual in the oldCourse, if not, save the examination in the changes
		=> 3. if there are changes on examination level, add the course with the name and
			  the examination changes to the slice of course changes
	*/
	courseDifferences := []crawler.Course{}
	for _, newCourse := range newCourses {
		examinationDifferences := []crawler.Examination{}
		for _, oldCourse := range oldCourses {
			if newCourse.Name == oldCourse.Name {
				for _, newExamination := range newCourse.Examinations {
					containsExamination := false
					for _, oldExamination := range oldCourse.Examinations {
						if reflect.DeepEqual(newExamination, oldExamination) {
							containsExamination = true
						}
					}
					if !containsExamination {
						examinationDifferences = append(examinationDifferences, newExamination)
					}
				}
			}
		}

		if len(examinationDifferences) > 0 {
			courseDifference := crawler.Course{Name: newCourse.Name, Examinations: examinationDifferences}
			courseDifferences = append(courseDifferences, courseDifference)
		}
	}
	return courseDifferences
}
