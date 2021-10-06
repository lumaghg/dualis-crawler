package database

import (
	"fmt"
	"log"
	"lumaghg/dualis-crawler/crawler"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/ses"
)

type App struct {
	dynamoClient *dynamodb.DynamoDB
	tableName    string
	email        string
}

type DynamoCourse struct {
	Email   string
	Courses []crawler.Course
}

func CompareAndUpdateCourses(courses []crawler.Course, email string) ([]crawler.Course, error) {
	//create Session and Client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(sess)

	gradesApp := App{dynamoClient: svc, tableName: "DUALIS_GRADES", email: email}

	dualisChanges, err := gradesApp.updateDatabaseAndGetChanges(courses)
	if err != nil {
		return []crawler.Course{}, err
	}
	fmt.Println("dualis changes: \n", dualisChanges)
	return dualisChanges, nil
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
		//if an item exists, replace it with the new, if not, create it
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

func SendUpdateEmail(dualisChanges []crawler.Course, notificationEmail string) error {
	fmt.Println("send email via aws")
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-central-1")},
	)
	if err != nil {
		return err
	}

	// Create an SES session.
	svc := ses.New(sess)
	//construct html email body
	htmlBody := "<h3>Folgende Module haben neue Bewertungen:</h3><br>"
	for _, course := range dualisChanges {
		htmlBody = htmlBody + course.Name + ": <br>"
		for _, examination := range course.Examinations {
			htmlBody = htmlBody + examination.Exam_type + ": " + examination.Grade + "<br>"
		}
	}
	htmlBody = htmlBody + "\nVielen Dank für dein Vertrauen in den Dualis-Bot!"

	//construct text email body
	textBody := "Folgende Module haben neue Bewertungen:\n\n"
	for _, course := range dualisChanges {
		textBody = textBody + course.Name + ":\n"
		for _, examination := range course.Examinations {
			textBody = textBody + examination.Exam_type + ": " + examination.Grade
		}
	}
	textBody = textBody + "\nVielen Dank für dein Vertrauen in den Dualis-Bot!"
	subject := "Es sind neue Bewertungen in Dualis verfügbar!"

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				//! Change, when the credentials table is ready
				aws.String(notificationEmail),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(htmlBody),
				},
				Text: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(textBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String("dualis-update@robin-reyer.de"),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	result, err := svc.SendEmail(input)

	// Display error messages if they occur.
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}

		return nil
	}

	fmt.Println("Email Sent to address: " + "robin.reyer@t-online.de")
	fmt.Println(result)
	return nil
}
