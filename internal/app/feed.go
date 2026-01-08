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

func (a *App) ListFeedTopicStatuses() ([]*models.TopicStatus, error) {
	service, err := servicefeed.GetTopicService()
	if err != nil {
		return nil, err
	}
	return service.ListTopicStatuses(), nil
}

func (a *App) GetTotalUnreadCount() (int, error) {
	service, err := servicefeed.GetTopicService()
	if err != nil {
		return 0, err
	}
	statuses := service.ListTopicStatuses()
	total := 0
	for _, s := range statuses {
		total += s.UnreadCount
	}
	return total, nil
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

func (a *App) MarkFeedItemAsRead(topic, id string) error {
	service, err := servicefeed.GetTopicService()
	if err != nil {
		return err
	}
	return service.MarkItemAsRead(topic, id)
}
