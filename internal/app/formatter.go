package app

import (
	"encoding/json"
	"regexp"

	"github.com/zjregee/alter/internal/models"
)

var (
	hanToLatin          = regexp.MustCompile(`([\p{Han}])([A-Za-z0-9])`)
	latinToHan          = regexp.MustCompile(`([A-Za-z0-9])([\p{Han}])`)
	hanToLatinMidPunct  = regexp.MustCompile(`([\p{Han}])([-/]+)([A-Za-z0-9])`)
	latinToHanMidPunct  = regexp.MustCompile(`([A-Za-z0-9])([-/]+)([\p{Han}])`)
	hanToLatinOpenPunct = regexp.MustCompile(`([\p{Han}])([\(\[\{'""]+)([A-Za-z0-9])`)
	latinToHanOpenPunct = regexp.MustCompile(`([A-Za-z0-9])([\(\[\{'""]+)([\p{Han}])`)
	hanToLatinPunct     = regexp.MustCompile(`([\p{Han}])([,.;:!?\)\]\}]+)([A-Za-z0-9])`)
	latinToHanPunct     = regexp.MustCompile(`([A-Za-z0-9])([,.;:!?\)\]\}]+)([\p{Han}])`)
)

func formatThreadMessage(content string) string {
	if content == "" {
		return content
	}

	content = hanToLatinMidPunct.ReplaceAllString(content, "$1 $2 $3")
	content = latinToHanMidPunct.ReplaceAllString(content, "$1 $2 $3")
	content = hanToLatinOpenPunct.ReplaceAllString(content, "$1 $2$3")
	content = latinToHanOpenPunct.ReplaceAllString(content, "$1 $2$3")
	content = hanToLatinPunct.ReplaceAllString(content, "$1$2 $3")
	content = latinToHanPunct.ReplaceAllString(content, "$1$2 $3")
	content = hanToLatin.ReplaceAllString(content, "$1 $2")
	content = latinToHan.ReplaceAllString(content, "$1 $2")

	return content
}

func formatThreadTitle(title string) string {
	return formatThreadMessage(title)
}

func formatThreadEvent(event *models.MarshaledThreadMessage) *models.MarshaledThreadMessage {
	formatted := &models.MarshaledThreadMessage{
		Type:    event.Type,
		Content: event.Content,
	}

	switch event.Type {
	case models.AgentMessageTypeThought:
		var msg models.AgentThought
		if err := json.Unmarshal(event.Content, &msg); err != nil {
			return formatted
		}
		msg.Content = formatThreadMessage(msg.Content)
		if content, err := json.Marshal(msg); err == nil {
			formatted.Content = content
		}
	case models.AgentMessageTypeError:
		var msg models.AgentError
		if err := json.Unmarshal(event.Content, &msg); err != nil {
			return formatted
		}
		msg.Error = formatThreadMessage(msg.Error)
		if content, err := json.Marshal(msg); err == nil {
			formatted.Content = content
		}
	default:
	}

	return formatted
}

func formatThreadTurns(turns []*models.ThreadMessageTurn) []*models.ThreadMessageTurn {
	formatted := make([]*models.ThreadMessageTurn, 0, len(turns))
	for _, turn := range turns {
		newTurn := &models.ThreadMessageTurn{
			Role:   turn.Role,
			Events: make([]*models.MarshaledThreadMessage, 0, len(turn.Events)),
		}

		for _, event := range turn.Events {
			newTurn.Events = append(newTurn.Events, formatThreadEvent(event))
		}

		formatted = append(formatted, newTurn)
	}

	return formatted
}
