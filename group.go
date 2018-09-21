package commander

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	uuid "github.com/satori/go.uuid"
)

const (
	// ParentHeader kafka message parent header
	ParentHeader = "parent"
	// ActionHeader kafka message action header
	ActionHeader = "action"
	// IDHeader kafka message id header
	IDHeader = "key"
	// AcknowledgedHeader kafka message acknowledged header
	AcknowledgedHeader = "acknowledged"
	// VersionHeader kafka message version header
	VersionHeader = "version"
)

// EventTopic contains the config information of a events topic
type EventTopic struct {
	Name string
}

// CommandTopic contains the config information of a commands topic
type CommandTopic struct {
	Name string
}

// Topic interface is used to store events and commands topics
type Topic interface{}

// Group contains information about a commander group.
// A commander group could contain a events and commands topic where
// commands and events could be consumed and produced to.
type Group struct {
	*Commander
	Timeout time.Duration
	Topics  []Topic
}

// AsyncCommand creates a command message to the given group command topic
// and does not await for the responding event.
//
// If no command key is set will the command id be used. A command key is used
// to write a command to the right kafka partition therefor to guarantee the order
// of the kafka messages is it important to define a "dataset" key.
func (group *Group) AsyncCommand(command *Command) ([]CommandTopic, error) {
	topics := []CommandTopic{}

	for _, t := range group.Topics {
		switch topic := t.(type) {
		case CommandTopic:
			err := group.ProduceCommand(command, topic.Name)
			if err != nil {
				return nil, err
			}

			topics = append(topics, topic)
		}
	}

	return topics, nil
}

// SyncCommand creates a command message to the given group and awaits
// its responding message. If no message is received within the set timeout period
// will a timeout be thrown.
func (group *Group) SyncCommand(command *Command) ([]*Event, error) {
	topics, err := group.AsyncCommand(command)
	if err != nil {
		return nil, err
	}

	// The timeout channel gets closed when all events are received within the timeout period
	sink, timeout := group.AwaitEvents(group.Timeout, command.ID, len(topics))
	defer close(sink)

	for message := range timeout {
		return nil, message
	}

	events := []*Event{}
	for event := range sink {
		events = append(events, event)
	}

	return events, err
}

// AwaitEvents awaits till the expected events are created with the given parent id.
// The returend events are buffered in the sink channel.
//
// If not the expected events are returned within the given timeout period
// will a error be returned. The timeout channel is closed when all
// expected events are received or after a timeout is thrown.
func (group *Group) AwaitEvents(timeout time.Duration, parent uuid.UUID, expecting int) (chan *Event, chan error) {
	sink := make(chan *Event, expecting)
	err := make(chan error, 1)

	events, closing := group.NewEventsConsumer()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()
	defer closing()
	defer close(err)

	go func() {
		// Wait for the events to return
		// A error is thrown if the event does not return within the given timeout period
	syncEvent:
		for {
			select {
			case event := <-events:
				if event.Parent != parent {
					continue
				}

				sink <- event
				expecting = expecting - 1

				if expecting <= 0 {
					break syncEvent
				}
			case <-ctx.Done():
				err <- errors.New("timeout reached")
				break
			}
		}
	}()

	return sink, err
}

// NewEvent creates a new event for the given group.
func (group *Group) NewEvent() {
}

// ProduceCommand constructs and produces a command kafka message to the given topic.
// A error is returned if anything went wrong in the process.
//
// If no command key is set will the command id be used. A command key is used
// to write a command to the right kafka partition therefor to guarantee the order
// of the kafka messages is it important to define a "dataset" key.
func (group *Group) ProduceCommand(command *Command, topic string) error {
	if command.Key == uuid.Nil {
		command.Key = command.ID
	}

	message := sarama.ProducerMessage{
		Headers: []sarama.RecordHeader{
			sarama.RecordHeader{
				Key:   []byte(ActionHeader),
				Value: []byte(command.Action),
			},
			sarama.RecordHeader{
				Key:   []byte(IDHeader),
				Value: []byte(command.ID.String()),
			},
		},
		Key:   sarama.StringEncoder(command.Key.String()),
		Value: sarama.ByteEncoder(command.Data),
		Topic: topic,
	}

	return group.Produce(&message)
}

// ProduceEvent produces a event kafka message to the given topic.
// A error is returned if anything went wrong in the process.
func (group *Group) ProduceEvent(event *Event, topic string) error {
	if event.Key == uuid.Nil {
		event.Key = event.ID
	}

	message := &sarama.ProducerMessage{
		Headers: []sarama.RecordHeader{
			sarama.RecordHeader{
				Key:   []byte(ActionHeader),
				Value: []byte(event.Action),
			},
			sarama.RecordHeader{
				Key:   []byte(ParentHeader),
				Value: []byte(event.Parent.String()),
			},
			sarama.RecordHeader{
				Key:   []byte(IDHeader),
				Value: []byte(event.ID.String()),
			},
			sarama.RecordHeader{
				Key:   []byte(AcknowledgedHeader),
				Value: []byte(strconv.FormatBool(event.Acknowledged)),
			},
			sarama.RecordHeader{
				Key:   []byte(VersionHeader),
				Value: []byte(strconv.Itoa(event.Version)),
			},
		},
		Key:   sarama.StringEncoder(event.Key.String()),
		Value: sarama.ByteEncoder(event.Data),
		Topic: topic,
	}

	return group.Produce(message)
}

// NewEventsConsumer starts consuming events of the given action and the given versions.
// The events topic used is set during initialization of the group.
// Two arguments are returned, a events channel and a method to unsubscribe the consumer.
// All received events are published over the returned events go channel.
func (group *Group) NewEventsConsumer() (chan *Event, func()) {
	sink := make(chan *Event, 1)

	go func() {

	}()

	return sink, func() {
	}
}

// NewCommandsConsumer starts consuming commands of the given action.
// The commands topic used is set during initialization of the group.
// Two arguments are returned, a events channel and a method to unsubscribe the consumer.
// All received events are published over the returned events go channel.
func (group *Group) NewCommandsConsumer(action string) (chan *Command, func()) {
	sink := make(chan *Command, 1)

	go func() {
		// Create a new command handle subscription for the given group topics
	}()

	return sink, func() {
	}
}

// EventHandle is a callback function used to handle/process events
type EventHandle func(*Event)

// NewEventsHandle is a small wrapper around NewEventsConsumer but calls the given callback method instead.
// Once a event of the given action is received is the EventHandle callback called.
// The handle is closed once the consumer receives a close signal.
func (group *Group) NewEventsHandle(action string, versions []int, callback EventHandle) func() {
	// sink := make(chan *Event, 1)

	go func() {
		// Create a new event handle subscription for the given group topic
	}()

	return func() {
	}
}

// CommandHandle is a callback function used to handle/process commands
type CommandHandle func(*Event)

// NewCommandsHandle is a small wrapper around NewCommandsConsumer but calls the given callback method instead.
// Once a event of the given action is received is the EventHandle callback called.
// The handle is closed once the consumer receives a close signal.
func (group *Group) NewCommandsHandle(action string, callback CommandHandle) func() {
	return nil
}