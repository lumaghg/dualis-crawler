package email

import (
	"fmt"
	"lumaghg/dualis-crawler/crawler"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

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
		htmlBody = htmlBody + "<br>" + course.Name + ": <br>"
		for _, examination := range course.Examinations {
			htmlBody = htmlBody + examination.Exam_type + ": <b>" + examination.Grade + "</b><br>"
		}
	}
	htmlBody = htmlBody + "<br>Vielen Dank für dein Vertrauen in den Dualis-Bot!"

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
