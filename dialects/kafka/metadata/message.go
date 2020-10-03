package metadata

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/jeroenrinzema/commander"
	"github.com/jeroenrinzema/commander/internal/metadata"
	"github.com/jeroenrinzema/commander/internal/types"
)

// MessageFromMessage attempts to construct a commander message of the given sarama consumer message
func MessageFromMessage(consumed *sarama.ConsumerMessage) *commander.Message {
	ctx := context.Background()

	ctx = NewKafkaContext(ctx, Kafka{
		Offset:    consumed.Offset,
		Partition: consumed.Partition,
	})

	message := &types.Message{
		Topic:     types.NewTopic(consumed.Topic, nil, types.EventMessage, types.DefaultMode),
		Data:      consumed.Value,
		Key:       consumed.Key,
		Timestamp: consumed.Timestamp,
	}

	message.NewCtx(ctx)
	headers := map[string][]string{}

headers:
	for _, record := range consumed.Headers {
		key := string(record.Key)

		switch key {
		case HeaderID:
			message.ID = string(record.Value)
			break
		case HeaderAction:
			message.Action = string(record.Value)
			break
		case HeaderStatusCode:
			status, err := strconv.ParseInt(string(record.Value), 10, 16)
			if err != nil {
				continue headers
			}

			message.Status = types.StatusCode(status)
			break
		case HeaderVersion:
			version, err := strconv.ParseInt(string(record.Value), 10, 8)
			if err != nil {
				continue headers
			}

			message.Version = types.Version(version)
			break
		case HeaderEOS:
			message.EOS = message.EOS.Parse(string(record.Value))
			break
		case HeaderParentID:
			message.NewCtx(metadata.NewParentIDContext(message.Ctx(), metadata.ParentID(record.Value)))
			break
		case HeaderParentTimestamp:
			unix, err := strconv.ParseInt(string(record.Value), 10, 64)
			if err != nil {
				continue headers
			}

			time := metadata.ParentTimestamp(time.Unix(0, unix))
			message.NewCtx(metadata.NewParentTimestampContext(message.Ctx(), time))
			break
		default:
			headers[key] = strings.Split(string(record.Value), metadata.HeaderValueDevider)
			break
		}
	}

	return message
}

// MessageToMessage attempts to construct a sarama consumer message of the given commander message.
func MessageToMessage(produce *commander.Message) *sarama.ProducerMessage {
	headers := []sarama.RecordHeader{
		sarama.RecordHeader{
			Key:   []byte(HeaderID),
			Value: []byte(produce.ID),
		},
		sarama.RecordHeader{
			Key:   []byte(HeaderAction),
			Value: []byte(produce.Action),
		},
		sarama.RecordHeader{
			Key:   []byte(HeaderVersion),
			Value: []byte(strconv.Itoa(int(produce.Version))),
		},
		sarama.RecordHeader{
			Key:   []byte(HeaderEOS),
			Value: []byte(produce.EOS.String()),
		},
		sarama.RecordHeader{
			Key:   []byte(HeaderStatusCode),
			Value: []byte(strconv.Itoa(int(produce.Status))),
		},
	}

	parent, has := metadata.ParentIDFromContext(produce.Ctx())
	if has {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte(HeaderParentID),
			Value: []byte(parent),
		})
	}

	timestamp, has := metadata.ParentTimestampFromContext(produce.Ctx())
	if has {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte(HeaderParentTimestamp),
			Value: []byte(strconv.Itoa(int(time.Time(timestamp).UnixNano()))),
		})
	}

	kv, has := metadata.HeaderFromContext(produce.Ctx())
	if has {
		for key, value := range kv {
			headers = append(headers, sarama.RecordHeader{
				Key:   []byte(key),
				Value: []byte(value.String()),
			})
		}
	}

	return &sarama.ProducerMessage{
		Topic:   produce.Topic.Name(),
		Key:     sarama.ByteEncoder(produce.Key),
		Value:   sarama.ByteEncoder(produce.Data),
		Headers: headers,
	}
}
