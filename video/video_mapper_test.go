package video

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	. "github.com/Financial-Times/upp-next-video-mapper/logger"
	"github.com/stretchr/testify/assert"
)

const (
	testUuid         = "bad50c54-76d9-30e9-8734-b999c708aa4c"
	messageTimestamp = "2017-04-13T10:27:32.353Z"
	xRequestId       = "tid_123123"
)

var mapper = VideoMapper{}

func init() {
	InitLogs(os.Stdout, os.Stdout, os.Stderr)
}

func TestTransformMsg_TidHeaderMissing(t *testing.T) {
	var message = consumer.Message{
		Headers: map[string]string{
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{}`,
	}

	_, _, err := mapper.TransformMsg(message)
	assert.EqualError(t, err, "X-Request-Id not found in kafka message headers. Skipping message", "Expected error when X-Request-Id is missing")
}

func TestTransformMsg_MessageTimestampHeaderMissing(t *testing.T) {
	var message = consumer.Message{
		Headers: map[string]string{
			"X-Request-Id": xRequestId,
		},
		Body: `{
			"id": "77fff607-bc22-450d-8c5d-e26fe1f0dc7c" 
		}`,
	}

	msg, _, err := mapper.TransformMsg(message)
	assert.NoError(t, err, "Error not expected when Message-Timestamp header is missing")
	assert.NotEmpty(t, msg.Body, "Message body should not be empty")
	assert.Contains(t, msg.Body, "\"lastModified\":", "LastModified field should be generated if header value is missing")
}

func TestTransformMsg_InvalidJson(t *testing.T) {
	var message = consumer.Message{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{{
					"lastModified": "2017-04-04T14:42:58.920Z",
					"publishReference": "tid_123123",
					"type": "video",
					"id": "bad50c54-76d9-30e9-8734-b999c708aa4c"}`,
	}

	_, _, err := mapper.TransformMsg(message)
	assert.Error(t, err, "Expected error when invalid JSON for video content")
	assert.Contains(t, err.Error(), "Video JSON couldn't be unmarshalled. Skipping invalid JSON:", "Expected error message when invalid JSON for video content")
}

func TestTransformMsg_UuidMissing(t *testing.T) {
	var message = consumer.Message{
		Headers: map[string]string{
			"X-Request-Id": xRequestId,
		},
		Body: `{}`,
	}

	_, _, err := mapper.TransformMsg(message)
	assert.Error(t, err, "Expected error when video UUID is missing")
	assert.Contains(t, err.Error(), "Could not extract UUID from video message. Skipping invalid JSON:", "Expected error when video UUID is missing")
}

func TestTransformMsg_UnpublishEvent(t *testing.T) {
	var message = consumer.Message{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Message-Timestamp": messageTimestamp,
		},
		Body: `{
					"deleted": true,
					"lastModified": "2017-04-04T14:42:58.920Z",
					"publishReference": "tid_123123",
					"type": "video",
					"id": "bad50c54-76d9-30e9-8734-b999c708aa4c"}`,
	}

	resultMsg, uuid, err := mapper.TransformMsg(message)
	assert.NoError(t, err, "Error not expected for unpublish event")
	assert.Equal(t, "bad50c54-76d9-30e9-8734-b999c708aa4c", uuid, "UUID not extracted correctly from unpublish event")
	assert.Equal(t, "{\"contentUri\":\"http://next-video-mapper.svc.ft.com/video/model/bad50c54-76d9-30e9-8734-b999c708aa4c\",\"payload\":{},\"lastModified\":\"2017-04-13T10:27:32.353Z\"}", resultMsg.Body)
}

func TestTransformMsg_Success(t *testing.T) {
	videoInput, err := readContent("video-input.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Input data for test cannot be loaded from external file")
	}
	videoOutput, err := readContent("video-output.json")
	if err != nil {
		assert.FailNow(t, err.Error(), "Output data for test cannot be loaded from external file")
	}

	var message = consumer.Message{
		Headers: map[string]string{
			"X-Request-Id":      xRequestId,
			"Message-Timestamp": messageTimestamp,
		},
		Body: videoInput,
	}

	resultMsg, _, err := mapper.TransformMsg(message)
	assert.NoError(t, err, "Error not expected for publish event")
	assert.Equal(t, videoOutput, resultMsg.Body)
}

func readContent(fileName string) (string, error) {
	data, err := ioutil.ReadFile("test-resources/" + fileName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s", data), nil
}
