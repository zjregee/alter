package feed

import (
	"context"
	"fmt"
	"strings"

	"github.com/zjregee/alter/internal/models"
	feedService "github.com/zjregee/alter/internal/service/feed"
)

func PushFeed(ctx context.Context, params *PushFeedParams) (string, error) {
	if params == nil {
		return "", fmt.Errorf("params must be provided")
	}

	topic := strings.TrimSpace(params.Topic)
	if topic == "" {
		return "", fmt.Errorf("topic must be provided")
	}

	item := &models.FeedItem{
		Topic:   topic,
		Title:   strings.TrimSpace(params.Title),
		Content: params.Content,
	}

	service, err := feedService.GetTopicService()
	if err != nil {
		return "", err
	}

	stored, err := service.PushItem(item)
	if err != nil {
		return "", err
	}

	return formatPushResult(stored), nil
}

func formatPushResult(item *models.FeedItem) string {
	if item == nil {
		return "Action: push_feed\nResult: (empty)"
	}

	var b strings.Builder
	fmt.Fprint(&b, "Action: push_feed")
	fmt.Fprintf(&b, "\nTopic: %s", item.Topic)
	if item.ID != "" {
		fmt.Fprintf(&b, "\nID: %s", item.ID)
	}
	if item.Title != "" {
		fmt.Fprintf(&b, "\nTitle: %s", item.Title)
	}
	if item.CreatedAt != 0 {
		fmt.Fprintf(&b, "\nCreated At: %d", item.CreatedAt)
	}
	if item.Content != "" {
		fmt.Fprint(&b, "\nContent:\n```text\n")
		fmt.Fprint(&b, strings.TrimRight(item.Content, "\n"))
		fmt.Fprint(&b, "\n```")
	}

	return b.String()
}
