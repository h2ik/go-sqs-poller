# sqs-queue
GoLang SQS Queue Worker using the AWS SDK

This is based off of [golang-sqs-worker-example](https://github.com/nabeken/golang-sqs-worker-example) but it uses the [official AWS golang SDK](https://github.com/aws/aws-sdk-go).

When running this make sure that you specify `AWS_SDK_LOAD_CONFIG=true` before running.  It will pick up credentials from the `~/.aws/config` file.

### To Do
- Add support for passing in access keys when creating the sqs service instead of auto loading them.