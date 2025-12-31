package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/zjregee/alter/internal/models"
)

const feedKeyPrefix = "feed:topic:"

func SaveFeedItem(item *models.FeedItem) error {
	if item == nil {
		return fmt.Errorf("feed item is required")
	}
	if item.Topic == "" {
		return fmt.Errorf("feed item topic is required")
	}
	if item.ID == "" {
		return fmt.Errorf("feed item id is required")
	}

	key, err := feedItemKey(item.Topic, item.ID)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal feed item %s: %w", item.ID, err)
	}

	return Put([]byte(key), payload)
}

func LoadFeedItemsByTopic(topic string) ([]*models.FeedItem, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	encodedTopic, err := encodeFeedTopic(topic)
	if err != nil {
		return nil, err
	}

	prefix := feedKeyPrefix + encodedTopic + ":"
	entries, err := List([]byte(prefix))
	if err != nil {
		return nil, err
	}

	items := make([]*models.FeedItem, 0, len(entries))
	for key, value := range entries {
		if len(value) == 0 {
			continue
		}

		var stored models.FeedItem
		if err := json.Unmarshal(value, &stored); err != nil {
			return nil, fmt.Errorf("failed to unmarshal feed item %s: %w", key, err)
		}
		items = append(items, &stored)
	}

	sortFeedItems(items)
	return items, nil
}

func LoadAllFeedItems() ([]*models.FeedItem, error) {
	entries, err := List([]byte(feedKeyPrefix))
	if err != nil {
		return nil, err
	}

	items := make([]*models.FeedItem, 0, len(entries))
	for key, value := range entries {
		if len(value) == 0 {
			continue
		}

		var stored models.FeedItem
		if err := json.Unmarshal(value, &stored); err != nil {
			return nil, fmt.Errorf("failed to unmarshal feed item %s: %w", key, err)
		}
		items = append(items, &stored)
	}

	sortFeedItems(items)
	return items, nil
}

func ListFeedTopics() ([]string, error) {
	entries, err := List([]byte(feedKeyPrefix))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for key := range entries {
		trimmed := strings.TrimPrefix(key, feedKeyPrefix)
		prefix, _, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		decoded, err := decodeFeedTopic(prefix)
		if err != nil {
			return nil, err
		}
		seen[decoded] = struct{}{}
	}

	topics := make([]string, 0, len(seen))
	for topic := range seen {
		topics = append(topics, topic)
	}
	sort.Strings(topics)
	return topics, nil
}

func feedItemKey(topic, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("feed item id is required")
	}
	encodedTopic, err := encodeFeedTopic(topic)
	if err != nil {
		return "", err
	}
	return feedKeyPrefix + encodedTopic + ":" + id, nil
}

func encodeFeedTopic(topic string) (string, error) {
	if topic == "" {
		return "", fmt.Errorf("topic is required")
	}
	return base64.RawURLEncoding.EncodeToString([]byte(topic)), nil
}

func decodeFeedTopic(encoded string) (string, error) {
	if encoded == "" {
		return "", fmt.Errorf("encoded topic is required")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode topic: %w", err)
	}
	return string(decoded), nil
}

func sortFeedItems(items []*models.FeedItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
}
