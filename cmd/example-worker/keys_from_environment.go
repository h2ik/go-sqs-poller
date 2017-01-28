package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/jwhitcraft/sqs-queue/worker"
)

func main() {
	// Make sure that AWS_SDK_LOAD_CONFIG=true is defined as an environment variable before running the application
	// like this

	// create the new client and return the url
	svc, url := worker.NewSQSClient("go-webhook-queue-test")
	// set the queue url
	worker.QueueURL = url
	// start the worker
	worker.Start(svc, worker.HandlerFunc(func (msg *sqs.Message) error {
		fmt.Println(aws.StringValue(msg.Body))
		return nil
	}))
}