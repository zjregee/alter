package models

type FeedItem struct {
	ID        string `json:"id"`
	Topic     string `json:"topic"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
	IsRead    bool   `json:"is_read"`
}

type TopicStatus struct {
	Name        string `json:"name"`
	UnreadCount int    `json:"unread_count"`
}
