package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedSqsClient struct {
	Config       *aws.Config
	Messages     []*sqs.Message
	ReceiveIndex int
	Cancel       context.CancelFunc
	QueueAPI
	mock.Mock
}

func (c *mockedSqsClient) GetQueueUrl(urlInput *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	url := fmt.Sprintf("https://sqs.%v.amazonaws.com/123456789/%v", *c.Config.Region, *urlInput.QueueName)

	return &sqs.GetQueueUrlOutput{QueueUrl: &url}, nil
}

func (c *mockedSqsClient) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	c.Called(input)

	startRange := c.ReceiveIndex
	endRange := startRange + rand.Intn(9)
	if endRange > totalNumberOfMessages {
		endRange = totalNumberOfMessages
		c.Cancel()
	}

	messages := c.Messages[startRange:endRange]
	c.ReceiveIndex += endRange - startRange

	return &sqs.ReceiveMessageOutput{Messages: messages}, nil
}

func (c *mockedSqsClient) DeleteMessage(input *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	c.Called(input)
	return &sqs.DeleteMessageOutput{}, nil
}

type mockedHandler struct {
	mock.Mock
}

func (mh *mockedHandler) HandleMessage(receiptHandle, foo, qux string) {
	mh.Called(receiptHandle, foo, qux)
}

type sqsEvent struct {
	Foo string `json:"foo"`
	Qux string `json:"qux"`
}

const maxNumberOfMessages = 9
const waitTimeSecond = 19

const totalNumberOfMessages = 100

func TestStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	region := "eu-west-1"
	awsConfig := &aws.Config{Region: &region}
	workerConfig := &Config{
		MaxNumberOfMessage: maxNumberOfMessages,
		QueueName:          "my-sqs-queue",
		WaitTimeSecond:     waitTimeSecond,
	}

	clientParams := buildClientParams()
	sqsMessages := make([]*sqs.Message, totalNumberOfMessages)
	for i := 0; i < totalNumberOfMessages; i++ {
		sqsMessages[i] = &sqs.Message{
			Body:          aws.String(`{ "foo": "bar", "qux": "baz" }`),
			ReceiptHandle: aws.String(strconv.Itoa(i)),
		}
	}
	client := &mockedSqsClient{Messages: sqsMessages, Config: awsConfig, Cancel: cancel}

	worker := New(client, workerConfig)

	handler := new(mockedHandler)
	handlerFunc := HandlerFunc(func(msg *sqs.Message) (err error) {
		event := &sqsEvent{}

		json.Unmarshal([]byte(aws.StringValue(msg.Body)), event)

		handler.HandleMessage(*msg.ReceiptHandle, event.Foo, event.Qux)

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
		client.On("ReceiveMessage", clientParams).Return()
		for i := 0; i < totalNumberOfMessages; i++ {
			receiptHandle := strconv.Itoa(i)
			client.On("DeleteMessage",
				&sqs.DeleteMessageInput{QueueUrl: clientParams.QueueUrl, ReceiptHandle: &receiptHandle}).Return().Once()
			handler.On("HandleMessage", receiptHandle, "bar", "baz").Return().Once()
		}

		worker.Start(ctx, handlerFunc)

		client.AssertExpectations(t)
		handler.AssertExpectations(t)
	})
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
