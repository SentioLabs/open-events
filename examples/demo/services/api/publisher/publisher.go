package publisher

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Publisher abstracts the SQS-publish operation so handlers can be unit-tested
// against a fake.
type Publisher interface {
	Publish(ctx context.Context, eventName string, body []byte, attrs map[string]string) (messageID string, err error)
}

// SQSPublisher publishes to an SQS queue via aws-sdk-go-v2. The endpoint
// comes from AWS_ENDPOINT_URL when set; otherwise real AWS.
type SQSPublisher struct {
	Client   *sqs.Client
	QueueURL string
}

// Publish sends the body to SQS with the given attributes as SQS message
// attributes. The eventName parameter is informational; callers must also
// place it in attrs under AttrEventName for downstream consumers.
func (p *SQSPublisher) Publish(ctx context.Context, eventName string, body []byte, attrs map[string]string) (string, error) {
	msgAttrs := make(map[string]sqstypes.MessageAttributeValue, len(attrs))
	for k, v := range attrs {
		msgAttrs[k] = sqstypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}
	}
	out, err := p.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:          aws.String(p.QueueURL),
		MessageBody:       aws.String(string(body)),
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
	EventName string
	Body      []byte
	Attrs     map[string]string
}

// Publish records the call and returns a deterministic fake message id.
func (f *FakePublisher) Publish(_ context.Context, eventName string, body []byte, attrs map[string]string) (string, error) {
	attrsCopy := make(map[string]string, len(attrs))
	for k, v := range attrs {
		attrsCopy[k] = v
	}
	f.Calls = append(f.Calls, FakeCall{
		EventName: eventName,
		Body:      append([]byte(nil), body...),
		Attrs:     attrsCopy,
	})
	return "fake-msg-id", nil
}
