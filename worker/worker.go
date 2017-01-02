package worker

import (
	"log"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type HandlerFunc func(msg *sqs.Message) error

func (f HandlerFunc) HandleMessage(msg *sqs.Message) error {
	return f(msg)
}

type Handler interface {
	HandleMessage(msg *sqs.Message) error
}

var (
	QueueURL string = ""
	MaxNumberOfMessage int64 = 10
	WaitTimeSecond int64 = 20
)

func Start(svc *sqs.SQS, h Handler) {
	for {
		log.Println("worker: Start polling")
		params := &sqs.ReceiveMessageInput{
			QueueUrl: aws.String(QueueURL), // Required
			MaxNumberOfMessages: aws.Int64(MaxNumberOfMessage),
			MessageAttributeNames: []*string{
				aws.String("All"), // Required
			},
			WaitTimeSeconds:         aws.Int64(WaitTimeSecond),
		}

		resp, err := svc.ReceiveMessage(params)
		if err != nil {
			log.Println(err)
			continue
		}
		if len(resp.Messages) > 0 {
			run(svc, h, resp.Messages)
		}
	}
}

// poll launches goroutine per received message and wait for all message to be processed
func run(svc *sqs.SQS, h Handler, messages []*sqs.Message) {
	numMessages := len(messages)
	log.Printf("worker: Received %d messages", numMessages)

	var wg sync.WaitGroup
	wg.Add(numMessages)
	for i := range messages {
		go func(m *sqs.Message) {
			// launch goroutine
			log.Println("worker: Spawned worker goroutine")
			defer wg.Done()
			if err := handleMessage(svc, m, h); err != nil {
				log.Println(err)
			}
		}(messages[i])
	}

	wg.Wait()
}

func handleMessage(svc *sqs.SQS, m *sqs.Message, h Handler) error {
	var err error
	err = h.HandleMessage(m)
	if err != nil {
		return err
	}

	params := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(QueueURL), // Required
		ReceiptHandle: m.ReceiptHandle, // Required
	}
	_, err = svc.DeleteMessage(params)
	if err != nil {
		return err
	}
	log.Printf("worker: deleted message from queue: %s", aws.StringValue(m.ReceiptHandle))

	return nil
}