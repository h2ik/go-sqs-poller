package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/stretchr/testify/mock"
)

type mockedSqsClient struct {
	Config   *aws.Config
	Response sqs.ReceiveMessageOutput
	sqsiface.SQSAPI
}

func (c *mockedSqsClient) GetQueueUrl(urlInput *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	url := fmt.Sprintf("https://sqs.%v.amazonaws.com/123456789/%v", c.Config.Region, urlInput.QueueName)

	return &sqs.GetQueueUrlOutput{QueueUrl: &url}, nil
}

func (c *mockedSqsClient) ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	return &c.Response, nil
}

func (c *mockedSqsClient) DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	c.Response = sqs.ReceiveMessageOutput{}

	return &sqs.DeleteMessageOutput{}, nil
}

type mockedHandler struct {
	mock.Mock
}

type sqsEvent struct {
	Foo string `json:"foo"`
	Qux string `json:"qux"`
}

func (mh *mockedHandler) HandleMessage(foo string, qux string) {
	mh.Called(foo, qux)
}

func TestStart(t *testing.T) {
	region := "eu-west-1"
	awsConfig := &aws.Config{Region: &region}
	workerConfig := &Config{QueueName: "my-sqs-queue"}
	client := setupMockedSqsClient(awsConfig)
	worker := New(client, workerConfig)

	ctx, cancel := contextAndCancel()
	defer cancel()

	handler := new(mockedHandler)
	handlerFunc := HandlerFunc(func(msg *sqs.Message) (err error) {
		event := &sqsEvent{}

		json.Unmarshal([]byte(aws.StringValue(msg.Body)), event)

		handler.HandleMessage(event.Foo, event.Qux)

		return
	})

	t.Run("when worker successfully receives a message", func(t *testing.T) {
		handler.On("HandleMessage", "bar", "baz").Return().Once()
		worker.Start(ctx, handlerFunc)

		handler.AssertExpectations(t)
	})
}

func contextAndCancel() (context.Context, context.CancelFunc) {
	delay := time.Now().Add(1 * time.Millisecond)

	return context.WithDeadline(context.Background(), delay)
}

func setupMockedSqsClient(awsConfig *aws.Config) sqsiface.SQSAPI {
	sqsMessage := &sqs.Message{Body: aws.String(`{ "foo": "bar", "qux": "baz" }`)}
	sqsResponse := sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{sqsMessage},
	}

	return &mockedSqsClient{Response: sqsResponse, Config: awsConfig}
}
