package models

type FeedItem struct {
	ID        string `json:"id"`
	Topic     string `json:"topic"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at"`
}
