package worker

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"go.elastic.co/apm/module/apmawssdkgo/v2"
)

// CreateSqsClient creates a client for SQS API
func CreateSqsClient(awsConfigs ...*aws.Config) sqsiface.SQSAPI {
	session := session.Must(session.NewSession())
	session = apmawssdkgo.WrapSession(session)

	return sqs.New(session, awsConfigs...)
}

func (config *Config) populateDefaultValues() {
	if config.MaxNumberOfMessage == 0 {
		config.MaxNumberOfMessage = 10
	}

	if config.WaitTimeSecond == 0 {
		config.WaitTimeSecond = 20
	}
}

func getQueueURL(ctx context.Context, client QueueAPI, queueName string) (queueURL string) {
	params := &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName), // Required
	}
	response, err := client.GetQueueUrlWithContext(ctx, params)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	queueURL = aws.StringValue(response.QueueUrl)

	return
}
