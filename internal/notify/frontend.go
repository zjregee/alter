package notify

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/zjregee/alter/internal/models"
)

func EmitAgentMessage(ctx context.Context, msgType string, content string) {
	if ctx == nil {
		return
	}

	runtime.EventsEmit(ctx, "agent:message", map[string]string{
		"type":    msgType,
		"content": content,
	})
}

func EmitAgentMessagesTruncated(ctx context.Context, threadID string, fromIndex int) {
	if ctx == nil {
		return
	}

	runtime.EventsEmit(ctx, "agent:messages_truncated", map[string]any{
		"thread_id":  threadID,
		"from_index": fromIndex,
	})
}

func EmitThreadTitleUpdated(ctx context.Context, threadID string, title string) {
	if ctx == nil {
		return
	}

	runtime.EventsEmit(ctx, "thread:title_updated", map[string]string{
		"thread_id": threadID,
		"title":     title,
	})
}

func EmitFeedItemPushed(ctx context.Context, item *models.FeedItem) {
	if ctx == nil || item == nil {
		return
	}

	runtime.EventsEmit(ctx, "feed:item_pushed", item)
}

func EmitSchedulerRunUpdated(ctx context.Context, run *models.ScheduleRun) {
	if ctx == nil || run == nil {
		return
	}

	runtime.EventsEmit(ctx, "scheduler:run_updated", run)
}
