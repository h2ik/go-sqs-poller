package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedSqsClient struct {
	Config   *aws.Config
	Response sqs.ReceiveMessageOutput
	QueueAPI
	mock.Mock
}

func (c *mockedSqsClient) GetQueueUrlWithContext(ctx aws.Context, urlInput *sqs.GetQueueUrlInput, options ...request.Option) (*sqs.GetQueueUrlOutput, error) {
	url := fmt.Sprintf("https://sqs.%v.amazonaws.com/123456789/%v", *c.Config.Region, *urlInput.QueueName)

	return &sqs.GetQueueUrlOutput{QueueUrl: &url}, nil
}

func (c *mockedSqsClient) ReceiveMessageWithContext(ctx aws.Context, input *sqs.ReceiveMessageInput, opts ...request.Option) (*sqs.ReceiveMessageOutput, error) {
	c.Called(ctx, input)

	return &c.Response, nil
}

func (c *mockedSqsClient) DeleteMessageWithContext(ctx aws.Context, input *sqs.DeleteMessageInput, opts ...request.Option) (*sqs.DeleteMessageOutput, error) {
	c.Called(ctx, input)
	c.Response = sqs.ReceiveMessageOutput{}

	return &sqs.DeleteMessageOutput{}, nil
}

type mockedHandler struct {
	mock.Mock
}

func (mh *mockedHandler) HandleMessage(foo string, qux string) {
	mh.Called(foo, qux)
}

type sqsEvent struct {
	Foo string `json:"foo"`
	Qux string `json:"qux"`
}

const maxNumberOfMessages = 1984
const waitTimeSecond = 1337

func TestStart(t *testing.T) {
	region := "eu-west-1"
	awsConfig := &aws.Config{Region: &region}
	workerConfig := &Config{
		MaxNumberOfMessage: maxNumberOfMessages,
		QueueName:          "my-sqs-queue",
		WaitTimeSecond:     waitTimeSecond,
	}

	clientParams := buildClientParams()
	sqsMessage := &sqs.Message{Body: aws.String(`{ "foo": "bar", "qux": "baz" }`), MessageId: aws.String("1234")}
	sqsResponse := sqs.ReceiveMessageOutput{Messages: []*sqs.Message{sqsMessage}}
	client := &mockedSqsClient{Response: sqsResponse, Config: awsConfig}
	deleteInput := &sqs.DeleteMessageInput{QueueUrl: clientParams.QueueUrl}

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

	t.Run("the worker has correct configuration", func(t *testing.T) {
		assert.Equal(t, worker.Config.QueueName, "my-sqs-queue", "QueueName has been set properly")
		assert.Equal(t, worker.Config.QueueURL, "https://sqs.eu-west-1.amazonaws.com/123456789/my-sqs-queue", "QueueURL has been set properly")
		assert.Equal(t, worker.Config.MaxNumberOfMessage, int64(maxNumberOfMessages), "MaxNumberOfMessage has been set properly")
		assert.Equal(t, worker.Config.WaitTimeSecond, int64(waitTimeSecond), "WaitTimeSecond has been set properly")
	})

	t.Run("the worker has correct default configuration", func(t *testing.T) {
		minimumConfig := &Config{
			QueueName: "my-sqs-queue",
		}
		worker := New(client, minimumConfig)

		assert.Equal(t, worker.Config.QueueName, "my-sqs-queue", "QueueName has been set properly")
		assert.Equal(t, worker.Config.QueueURL, "https://sqs.eu-west-1.amazonaws.com/123456789/my-sqs-queue", "QueueURL has been set properly")
		assert.Equal(t, worker.Config.MaxNumberOfMessage, int64(10), "MaxNumberOfMessage has been set by default")
		assert.Equal(t, worker.Config.WaitTimeSecond, int64(20), "WaitTimeSecond has been set by default")
	})

	t.Run("the worker successfully processes a message", func(t *testing.T) {
		client.On("ReceiveMessageWithContext", mock.Anything, clientParams).Return()
		client.On("DeleteMessageWithContext", mock.Anything, deleteInput).Return()
		handler.On("HandleMessage", "bar", "baz").Return().Once()

		worker.Start(ctx, handlerFunc)

		client.AssertExpectations(t)
		handler.AssertExpectations(t)
	})
}

func contextAndCancel() (context.Context, context.CancelFunc) {
	delay := time.Now().Add(1 * time.Millisecond)

	return context.WithDeadline(context.Background(), delay)
}

func buildClientParams() *sqs.ReceiveMessageInput {
	url := aws.String("https://sqs.eu-west-1.amazonaws.com/123456789/my-sqs-queue")

	return &sqs.ReceiveMessageInput{
		QueueUrl:            url,
		MaxNumberOfMessages: aws.Int64(maxNumberOfMessages),
		AttributeNames:      []*string{aws.String("All")},
		WaitTimeSeconds:     aws.Int64(waitTimeSecond),
	}
}
