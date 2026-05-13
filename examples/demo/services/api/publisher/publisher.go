package publisher

import (
	"context"
	"maps"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Publisher abstracts the SQS-publish operation so handlers can be unit-tested
// against a fake. attrs must contain AttrEventName so the consumer can route
// the message without decoding the body.
type Publisher interface {
	Publish(ctx context.Context, body string, attrs map[string]string) (messageID string, err error)
}

// SQSPublisher publishes to an SQS queue via aws-sdk-go-v2. The endpoint
// comes from AWS_ENDPOINT_URL when set; otherwise real AWS.
type SQSPublisher struct {
	Client   *sqs.Client
	QueueURL string
}

// Publish sends the body to SQS with the given attributes as SQS message
// attributes.
func (p *SQSPublisher) Publish(ctx context.Context, body string, attrs map[string]string) (string, error) {
	msgAttrs := make(map[string]sqstypes.MessageAttributeValue, len(attrs))
	for k, v := range attrs {
		msgAttrs[k] = sqstypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}
	out, err := p.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(p.QueueURL),
		MessageBody:       aws.String(body),
		MessageAttributes: msgAttrs,
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.MessageId), nil
}

// FakePublisher records publish calls for unit tests.
type FakePublisher struct {
	Calls []FakeCall
}

// FakeCall captures the arguments passed to FakePublisher.Publish.
type FakeCall struct {
	Body  string
	Attrs map[string]string
}

// Publish records the call and returns a deterministic fake message id.
func (f *FakePublisher) Publish(_ context.Context, body string, attrs map[string]string) (string, error) {
	f.Calls = append(f.Calls, FakeCall{
		Body:  body,
		Attrs: maps.Clone(attrs),
	})
	return "fake-msg-id", nil
}
