package worker

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// NewSQSClient returns a SQS Client and a Queue URL for you you to connect to
func NewSQSClient(queueName string) (*sqs.SQS, string) {
	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return nil, ""
	}
	svc := sqs.New(sess)
	// try and find the queue url

	params := &sqs.GetQueueUrlInput{
		QueueName:              aws.String(queueName), // Required
	}
	resp, err := svc.GetQueueUrl(params)

	if err != nil {
		// Print the error, cast err to aws err.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return nil, ""
	}

	return svc, aws.StringValue(resp.QueueUrl)
}