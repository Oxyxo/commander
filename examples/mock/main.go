package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jeroenrinzema/commander"
	uuid "github.com/satori/go.uuid"
)

/**
 * A commander group contains all the information needed for commander
 * to setup it's consumers and producers.
 */
var group = &commander.Group{
	Topics: []commander.Topic{
		commander.Topic{
			Name:    "commands",
			Type:    commander.CommandTopic,
			Consume: true,
			Produce: true,
		},
		commander.Topic{
			Name:    "events",
			Type:    commander.EventTopic,
			Consume: true,
			Produce: true,
		},
	},
	Timeout: 5 * time.Second,
}

func main() {
	connectionstring := ""
	dialect := &commander.MockDialect{}

	/**
	 * When constrcuting a new commander instance do you have to construct a commander.Dialect as well.
	 * A dialect consists mainly of a producer and a consumer that acts as a connector to the wanted infastructure.
	 */
	_, err := commander.New(dialect, connectionstring, group)
	if err != nil {
		panic(err)
	}

	/**
	 * HandleFunc handles an "example" command. Once a command with the action "example" is
	 * processed will a event with the action "created" be produced to the events topic.
	 */
	group.HandleFunc("example", commander.CommandTopic, func(writer commander.ResponseWriter, message interface{}) {
		writer.ProduceEvent("created", 1, uuid.NewV4(), nil)
	})

	/**
	 * Handle creates a new "example" command that is produced to the groups writable command topic.
	 * Once the command is written is a responding event awaited. The responsing event has a header
	 * with the parent id set to the id of the received command.
	 */
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		command := commander.NewCommand("example", uuid.NewV4(), nil)
		event, err := group.SyncCommand(command)

		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	})

	fmt.Println("Http server running at :8080")
	fmt.Println("Send a http request to / to simulate a 'sync' command")

	http.ListenAndServe(":8080", nil)
}