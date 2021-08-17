package worker

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// HandlerFunc is used to define the Handler that is run on for each message
type HandlerFunc func(msg *sqs.Message) error

// HandleMessage wraps a function for handling sqs messages
func (f HandlerFunc) HandleMessage(msg *sqs.Message) error {
	return f(msg)
}

// Handler interface
type Handler interface {
	HandleMessage(msg *sqs.Message) error
}

// InvalidEventError struct
type InvalidEventError struct {
	event string
	msg   string
}

func (e InvalidEventError) Error() string {
	return fmt.Sprintf("[Invalid Event: %s] %s", e.event, e.msg)
}

// NewInvalidEventError creates InvalidEventError struct
func NewInvalidEventError(event, msg string) InvalidEventError {
	return InvalidEventError{event: event, msg: msg}
}

// QueueAPI interface is the minimum interface required from a queue implementation to invoke New worker.
// Invoking worker.New() takes in a queue name which is why GetQueueUrl is needed.
type QueueAPI interface {
	GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error)
	QueueDeleteReceiverAPI
}

// QueueDeleteReceiverAPI interface is the minimum interface required to run a worker.
// When a worker is in its Receive loop, it requires this interface.
type QueueDeleteReceiverAPI interface {
	DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
}

// Worker struct
type Worker struct {
	Config    *Config
	Log       LoggerIFace
	SqsClient QueueDeleteReceiverAPI
}

// Config struct
type Config struct {
	MaxNumberOfMessage int64
	QueueName          string
	QueueURL           string
	WaitTimeSecond     int64
}

// New sets up a new Worker
func New(client QueueAPI, config *Config) *Worker {
	config.populateDefaultValues()
	config.QueueURL = getQueueURL(client, config.QueueName)

	return &Worker{
		Config:    config,
		Log:       &logger{},
		SqsClient: client,
	}
}

// Start starts the polling and will continue polling till the application is forcibly stopped
func (worker *Worker) Start(ctx context.Context, h Handler) {
	for {
		select {
		case <-ctx.Done():
			log.Println("worker: Stopping polling because a context kill signal was sent")
			return
		default:
			worker.Log.Debug("worker: Start Polling")

			params := &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(worker.Config.QueueURL), // Required
				MaxNumberOfMessages: aws.Int64(worker.Config.MaxNumberOfMessage),
				AttributeNames: []*string{
					aws.String("All"), // Required
				},
				MessageAttributeNames: []*string{
					aws.String(sqs.QueueAttributeNameAll),
				},
				WaitTimeSeconds: aws.Int64(worker.Config.WaitTimeSecond),
			}

			resp, err := worker.SqsClient.ReceiveMessage(params)
			if err != nil {
				log.Println(err)
				continue
			}
			if len(resp.Messages) > 0 {
				worker.run(h, resp.Messages)
			}
		}
	}
}

// poll launches goroutine per received message and wait for all message to be processed
func (worker *Worker) run(h Handler, messages []*sqs.Message) {
	numMessages := len(messages)
	worker.Log.Info(fmt.Sprintf("worker: Received %d messages", numMessages))

	var wg sync.WaitGroup
	wg.Add(numMessages)
	for i := range messages {
		go func(m *sqs.Message) {
			// launch goroutine
			defer wg.Done()
			if err := worker.handleMessage(m, h); err != nil {
				worker.Log.Error(err.Error())
			}
		}(messages[i])
	}

	wg.Wait()
}

func (worker *Worker) handleMessage(m *sqs.Message, h Handler) error {
	var err error
	err = h.HandleMessage(m)
	if _, ok := err.(InvalidEventError); ok {
		worker.Log.Error(err.Error())
	} else if err != nil {
		return err
	}

	params := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(worker.Config.QueueURL), // Required
		ReceiptHandle: m.ReceiptHandle,                    // Required
	}
	_, err = worker.SqsClient.DeleteMessage(params)
	if err != nil {
		return err
	}
	worker.Log.Debug(fmt.Sprintf("worker: deleted message from queue: %s", aws.StringValue(m.ReceiptHandle)))

	return nil
}
