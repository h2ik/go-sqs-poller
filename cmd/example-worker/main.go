package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/jwhitcraft/sqs-queue/worker"
)

func Print(msg *sqs.Message) error {
	fmt.Println(aws.StringValue(msg.Body))
	return nil
}

func main() {
	svc, url := worker.NewSQSClient("go-webhook-queue-test")
	// set the queue url
	worker.QueueURL = url
	// start the worker
	worker.Start(svc, worker.HandlerFunc(Print))
}