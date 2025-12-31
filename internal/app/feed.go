package app

import (
	"fmt"

	"github.com/zjregee/alter/internal/models"
	servicefeed "github.com/zjregee/alter/internal/service/feed"
)

func (a *App) ListFeedTopics() ([]string, error) {
	service, err := servicefeed.GetTopicService()
	if err != nil {
		return nil, err
	}

	return service.ListTopics(), nil
}

func (a *App) LoadFeedTopic(topic string) ([]*models.FeedItem, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	service, err := servicefeed.GetTopicService()
	if err != nil {
		return nil, err
	}

	return service.LoadTopic(topic)
}
