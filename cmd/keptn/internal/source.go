package internal

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
)

const (
	// PluginName is the name of the Keptn Botkube plugin.
	PluginName = "keptn"

	description = "Keptn plugin polls events from configured Keptn API endpoint."

	pollPeriodInSeconds = 5
)

var emojiForStatus = map[string]string{
	"succeeded": ":large_green_circle:",
	"errored":   ":x:",
	"aborted":   ":warning:",
	"":          ":email:",
}

// Source prometheus source plugin data structure
type Source struct {
	pluginVersion string
	config        Config
	eventCh       chan source.Event
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

// Stream streams Keptn events
func (p *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	config, err := MergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	s := Source{
		eventCh: make(chan source.Event),
		config:  config,
	}
	go consumeEvents(ctx, &s)
	return source.StreamOutput{
		Event: s.eventCh,
	}, nil
}

// Metadata returns metadata of Keptn configuration
func (p *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     p.pluginVersion,
		Description: description,
		JSONSchema:  jsonSchema(),
	}, nil
}

func consumeEvents(ctx context.Context, s *Source) {
	keptn, err := NewClient(s.config.URL, s.config.Token)
	exitOnError(err)

	for {
		req := GetEventsRequest{
			Project:  "botkube",
			FromTime: time.Now().Add(-time.Second * pollPeriodInSeconds),
		}
		res, err := keptn.Events(ctx, &req)
		if err != nil {
			log.Printf("failed to get events. %v", err)
		}
		for _, event := range res {
			message := source.Event{
				Message:         messageFrom(event),
				RawObject:       event,
				AnalyticsLabels: event.ToAnonymizedEventDetails(),
			}
			s.eventCh <- message
		}
		// Fetch events periodically with given frequency
		time.Sleep(time.Second * pollPeriodInSeconds)
	}
}

func messageFrom(event *Event) api.Message {
	emoji := ""
	if event.Data.Status != "" {
		emoji = emojiForStatus[event.Data.Status]
	}
	section := api.Section{
		Base: api.Base{
			Header: fmt.Sprintf("%s %s", emoji, event.Type),
		},
	}
	section.Body.Plaintext = bulletPointEventAttachments(event)

	return api.Message{
		Sections: []api.Section{section},
	}
}

func bulletPointEventAttachments(event *Event) string {
	strBuilder := strings.Builder{}
	var labels []string
	appendToListIfNotEmpty(&labels, "ID", event.ID)
	appendToListIfNotEmpty(&labels, "Source", event.Source)
	appendToListIfNotEmpty(&labels, "Message", event.Data.Message)
	writeStringIfNotEmpty(&strBuilder, "Labels", bulletPointListFromMessages(labels))
	return strBuilder.String()
}

func appendToListIfNotEmpty(msgs *[]string, title, in string) {
	if in == "" {
		return
	}

	*msgs = append(*msgs, fmt.Sprintf("%s: %s", title, in))
}

func writeStringIfNotEmpty(strBuilder *strings.Builder, title, in string) {
	if in == "" {
		return
	}

	strBuilder.WriteString(fmt.Sprintf("*%s:*\n%s", title, in))
}

func bulletPointListFromMessages(msgs []string) string {
	return joinMessages(msgs, "• ")
}

func joinMessages(msgs []string, msgPrefix string) string {
	if len(msgs) == 0 {
		return ""
	}

	var strBuilder strings.Builder
	for _, m := range msgs {
		strBuilder.WriteString(fmt.Sprintf("%s%s\n", msgPrefix, m))
	}

	return strBuilder.String()
}

func jsonSchema() api.JSONSchema {
	return api.JSONSchema{
		Value: heredoc.Docf(`
		  {
			"$schema": "http://json-schema.org/draft-04/schema#",
			"title": "Keptn",
			"description": "%s",
			"type": "object",
			"properties": {
			  "url": {
				"description": "Keptn API endpoint",
				"type": "string",
				"default": "http://localhost:8080/api"
			  },
			  "token": {
				"description": "Keptn API Token",
				"type": "string"
			  },
			  "project": {
				"description": "Keptn Project",
				"type": "string"
			  },
			  "service": {
				"description": "Keptn Service",
				"type": "string"
			  },
			},
			"required": []
		  }`, description),
	}
}

func exitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
