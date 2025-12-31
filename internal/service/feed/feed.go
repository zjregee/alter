package feed

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/zjregee/alter/internal/models"
	"github.com/zjregee/alter/internal/service/storage"
	"github.com/zjregee/alter/internal/utils"
)

type TopicService struct {
	mu     sync.RWMutex
	topics map[string][]*models.FeedItem
}

var (
	topicServiceInstance *TopicService
	topicServiceOnce     sync.Once
)

func GetTopicService() (*TopicService, error) {
	var initErr error
	topicServiceOnce.Do(func() {
		topicServiceInstance, initErr = newTopicService()
	})
	return topicServiceInstance, initErr
}

func newTopicService() (*TopicService, error) {
	items, err := storage.LoadAllFeedItems()
	if err != nil {
		return nil, err
	}

	service := &TopicService{
		topics: make(map[string][]*models.FeedItem),
	}

	for _, item := range items {
		if item == nil || item.Topic == "" {
			continue
		}
		service.topics[item.Topic] = append(service.topics[item.Topic], cloneFeedItem(item))
	}

	for topic := range service.topics {
		service.sortTopicLocked(topic)
	}

	return service, nil
}

func (s *TopicService) ListTopics() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	topics := make([]string, 0, len(s.topics))
	for topic := range s.topics {
		topics = append(topics, topic)
	}
	sort.Strings(topics)
	return topics
}

func (s *TopicService) LoadTopic(topic string) ([]*models.FeedItem, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	s.mu.RLock()
	items, ok := s.topics[topic]
	s.mu.RUnlock()

	if !ok {
		return []*models.FeedItem{}, nil
	}

	cloned := make([]*models.FeedItem, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, cloneFeedItem(item))
	}
	return cloned, nil
}

func (s *TopicService) PushItem(item *models.FeedItem) (*models.FeedItem, error) {
	if item == nil {
		return nil, fmt.Errorf("feed item is required")
	}
	if item.Topic == "" {
		return nil, fmt.Errorf("feed item topic is required")
	}
	if item.ID == "" {
		item.ID = utils.GenerateUUID()
	}
	if item.CreatedAt == 0 {
		item.CreatedAt = time.Now().UnixMilli()
	}

	stored := cloneFeedItem(item)
	if err := storage.SaveFeedItem(stored); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.topics[item.Topic] = append(s.topics[item.Topic], stored)
	s.sortTopicLocked(item.Topic)
	s.mu.Unlock()

	return cloneFeedItem(stored), nil
}

func (s *TopicService) sortTopicLocked(topic string) {
	items := s.topics[topic]
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	s.topics[topic] = items
}

func cloneFeedItem(item *models.FeedItem) *models.FeedItem {
	if item == nil {
		return nil
	}
	copied := *item
	return &copied
}
