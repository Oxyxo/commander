package types

import (
	"context"
	"time"
)

// MessageType represents a message type
type MessageType int8

// Available message types
const (
	EventMessage MessageType = iota + 1
	CommandMessage
)

const (
	// ParentHeader kafka message parent header
	ParentHeader = "cmdr_parent"
	// ActionHeader kafka message action header
	ActionHeader = "cmdr_action"
	// IDHeader kafka message id header
	IDHeader = "cmdr_id"
	// StatusHeader kafka message status header
	StatusHeader = "cmdr_status"
	// VersionHeader kafka message version header
	VersionHeader = "cmdr_version"
	// MetaHeader kafka message meta header
	MetaHeader = "cmdr_meta"
	// CommandTimestampHeader kafka message command timestamp header as UNIX
	CommandTimestampHeader = "cmdr_command_timestamp"
)

// Message a message
type Message struct {
	Topic     Topic             `json:"topic"`
	Headers   map[string]string `json:"headers"`
	Value     []byte            `json:"value"`
	Key       []byte            `json:"key"`
	Retries   int               `json:"retries"`
	Offset    int               `json:"offset"`
	Partition int               `json:"partition"`
	Timestamp time.Time         `json:"timestamp"`
	Ctx       context.Context   `json:"-"`
}
