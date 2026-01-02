package scheduler

import (
	"container/heap"
	"time"

	"github.com/zjregee/alter/internal/models"
)

var _ heap.Interface = (*scheduleQueue)(nil)

type scheduleQueueItem struct {
	schedule *models.Schedule
	nextRun  time.Time
	index    int
}

type scheduleQueue struct {
	items []*scheduleQueueItem
	index map[string]*scheduleQueueItem
}

func newScheduleQueue() *scheduleQueue {
	sq := &scheduleQueue{
		items: make([]*scheduleQueueItem, 0),
		index: make(map[string]*scheduleQueueItem),
	}
	heap.Init(sq)
	return sq
}

func (sq *scheduleQueue) Len() int {
	return len(sq.items)
}

func (sq *scheduleQueue) Less(i, j int) bool {
	return sq.items[i].nextRun.Before(sq.items[j].nextRun)
}

func (sq *scheduleQueue) Swap(i, j int) {
	sq.items[i], sq.items[j] = sq.items[j], sq.items[i]
	sq.items[i].index = i
	sq.items[j].index = j
}

func (sq *scheduleQueue) Push(x any) {
	item := x.(*scheduleQueueItem)
	item.index = len(sq.items)
	sq.items = append(sq.items, item)
	sq.index[item.schedule.ID] = item
}

func (sq *scheduleQueue) Pop() any {
	old := sq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	sq.items = old[0 : n-1]
	delete(sq.index, item.schedule.ID)
	return item
}

func (sq *scheduleQueue) Peek() *scheduleQueueItem {
	if len(sq.items) == 0 {
		return nil
	}
	return sq.items[0]
}

func (sq *scheduleQueue) Remove(scheduleID string) {
	item, exists := sq.index[scheduleID]
	if !exists {
		return
	}
	heap.Remove(sq, item.index)
}
