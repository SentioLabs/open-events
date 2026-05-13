//go:build integration

package publisher

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// TestSQSPublisher_RoundTrip exercises the real SQS publish path against a
// LocalStack queue. It is skipped when OPENEVENTS_QUEUE_URL is unset so the
// default unit-test command stays self-contained.
func TestSQSPublisher_RoundTrip(t *testing.T) {
	queueURL := os.Getenv("OPENEVENTS_QUEUE_URL")
	if queueURL == "" {
		t.Skip("OPENEVENTS_QUEUE_URL not set; start LocalStack and run `make seed`")
	}
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		t.Fatalf("aws config: %v", err)
	}
	client := sqs.NewFromConfig(cfg)
	p := &SQSPublisher{Client: client, QueueURL: queueURL}
	id, err := p.Publish(context.Background(), "hello", map[string]string{
		AttrEventName: "test.event@1",
		AttrSchema:    SchemaValue,
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if id == "" {
		t.Fatal("empty message id")
	}
}
