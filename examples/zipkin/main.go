package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"github.com/jeroenrinzema/commander"
	"github.com/jeroenrinzema/commander/dialects/mock"
	"github.com/jeroenrinzema/commander/middleware/zipkin"
)

// Available flags
var (
	ZipkinHost  = ""
	ServiceName = ""
)

func init() {
	commander.Logger.SetOutput(os.Stdout)

	flag.StringVar(&ZipkinHost, "host", "http://127.0.0.1:9411/api/v2/spans", "Zipkin host")
	flag.StringVar(&ServiceName, "name", "example", "Service name")
	flag.Parse()
}

func main() {
	dialect := mock.NewDialect()
	group := commander.NewGroup(
		commander.NewTopic("commands", dialect, commander.CommandMessage, commander.DefaultMode),
		commander.NewTopic("events", dialect, commander.EventMessage, commander.DefaultMode),
	)

	// The mock dialect does not throw any error thus could safely be ignored
	client, _ := commander.NewClient(group)

	zconnect := fmt.Sprintf("host=%s name=%s", ZipkinHost, ServiceName)
	tracing, err := zipkin.New(zconnect)
	if err != nil {
		panic(err)
	}

	client.Middleware.Use(tracing)

	/**
	 * HandleFunc handles an "example" command. Once a command with the action "example" is
	 * processed will a event with the action "created" be produced to the events topic.
	 */
	group.HandleFunc(commander.CommandMessage, "example", func(writer commander.ResponseWriter, message interface{}) {
		key := uuid.Must(uuid.NewV4()).Bytes()
		writer.ProduceEventEOS("created", 1, key, nil)
	})

	/**
	 * Handle creates a new "example" command that is produced to the groups writable command topic.
	 * Once the command is written is a responding event awaited. The responding event has a header
	 * with the parent id set to the id of the received command.
	 */
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		span := tracing.Tracer.StartSpan("http.sync.example")
		defer span.Finish()

		key := uuid.Must(uuid.NewV4()).Bytes()

		command := commander.NewCommand("example", 1, key, nil)
		command.Headers = zipkin.ConstructMessageHeaders(span.Context())
		event, next, err := group.SyncCommand(command)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		next(nil)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	})

	fmt.Println("Http server running at :8080")
	fmt.Println("Send a http request to / to simulate a 'sync' command")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
