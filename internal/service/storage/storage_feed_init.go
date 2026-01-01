package storage

import (
	"time"

	"github.com/zjregee/alter/internal/models"
)

func initFeedItems() {
	now := time.Now()
	items := []*models.FeedItem{
		{
			ID:    "seed-release-1",
			Topic: "Releases",
			Title: "Spring Update",
			Content: `This release focuses on speed and clarity. Page loads are noticeably faster, and the new status chips make it easier to scan what changed.

We reworked how the feed groups activity, so bursts of edits are tied together rather than scattered. The timeline now favors intent over raw time order.

If you are reviewing a long session, the jump links should keep you oriented without losing detail.

As always, share any cases where a feed item feels out of place so we can tune the grouping rules.`,
			CreatedAt: now.Add(-120 * time.Hour).UnixMilli(),
		},
		{
			ID:    "seed-release-2",
			Topic: "Releases",
			Title: "Tooling Refresh",
			Content: `The tool panel now supports clearer descriptions and more structured output. This makes it easier to read results, especially when a tool returns multiple fields.

We standardized labels, so similar tools use the same terms for inputs and outputs. That consistency should reduce the need to re-learn each panel.

We also prepared the pipeline for richer tool schemas, so future additions can include optional metadata without breaking existing behavior.

If you spot a tool that still feels verbose, call it out and we will tighten the layout.`,
			CreatedAt: now.Add(-72 * time.Hour).UnixMilli(),
		},
		{
			ID:    "seed-tips-1",
			Topic: "Tips",
			Title: "Write Sharper Topics",
			Content: `Short, specific topics read better in the feed. Think of a topic as a channel name, not a whole sentence.

Aim for two or three words that map to a habit. It helps readers scan and form a mental map quickly.

If you need detail, put it in the title and content. That separation keeps the timeline clean while still letting you tell the full story in each entry.

When in doubt, pick the broadest word that still feels accurate.`,
			CreatedAt: now.Add(-48 * time.Hour).UnixMilli(),
		},
		{
			ID:    "seed-tips-2",
			Topic: "Tips",
			Title: "Show, Then Tell",
			Content: `A good entry starts with one concrete detail before the explanation. A short lead line helps readers decide whether to open it.

Think of the first line as a preview of impact, not a summary. Numbers, dates, or a quick before and after work well.

Once opened, use two or three short paragraphs. It reads faster than a single dense block, and it is easier to skim on mobile.

If the entry includes actions, end with the next step so the reader knows what to do.`,
			CreatedAt: now.Add(-36 * time.Hour).UnixMilli(),
		},
		{
			ID:    "seed-alerts-1",
			Topic: "Alerts",
			Title: "Storage Maintenance Window",
			Content: `We will run a short maintenance window this weekend. During that time, new feed items may appear a few minutes later than usual.

The window is expected to last under an hour. We will post a follow up entry once the work is done.

No data will be removed. If you see a delay, it should resolve on its own once the maintenance finishes.

If anything looks stuck after the window, send us the item ID so we can check the queue.`,
			CreatedAt: now.Add(-24 * time.Hour).UnixMilli(),
		},
		{
			ID:    "seed-alerts-2",
			Topic: "Alerts",
			Title: "Index Rebuild Complete",
			Content: `The background rebuild finished successfully. Search should feel more consistent, especially when filtering by topic.

We also cleaned up a few older records that had duplicate timestamps. That should improve ordering in long running feeds.

If you notice anything odd, capture the item title and time so we can trace the record quickly.

Thanks for the patience while the rebuild ran in the background.`,
			CreatedAt: now.Add(-12 * time.Hour).UnixMilli(),
		},
		{
			ID:        "seed-guide-1",
			Topic:     "Guides",
			Title:     "Reading a Long Feed Entry",
			Content:   "# How to read a long entry\n\nThis is a longer example designed to stress the markdown renderer. It includes headings, lists, emphasis, and code blocks.\n\n## Quick scan checklist\n\n- **Title** tells you the outcome\n- _Timestamp_ tells you the context\n- First paragraph tells you the impact\n\n### Example snippet\n\nUse the output structure to reason about next steps:\n\n```json\n{\n  \"topic\": \"Guides\",\n  \"title\": \"Reading a Long Feed Entry\",\n  \"status\": \"ok\",\n  \"notes\": [\"scan\", \"open\", \"act\"]\n}\n```\n\n## Deep dive\n\nWhen you open a long entry, look for the first concrete detail: a number, a date, or a specific change. That detail anchors your understanding and helps you decide what to do next.\n\nIf the entry has multiple sections, skim the headings first, then jump to the part that affects your workflow. Do not read line by line unless you need the nuance.\n\n> Tip: If a section feels dense, pause and copy the key line into your task list. It saves time later.\n\n### Final note\n\nClose with a clear action: who owns it, what happens next, and when to follow up.",
			CreatedAt: now.Add(-8 * time.Hour).UnixMilli(),
		},
		{
			ID:        "seed-guide-2",
			Topic:     "Guides",
			Title:     "Markdown Stress Test",
			Content:   "# Markdown Stress Test\n\nThis entry mixes multiple markdown elements to verify rendering and layout behavior.\n\n## Section: Lists and tables\n\n1. First item with **bold** text\n2. Second item with _italic_ text\n3. Third item with `inline code`\n\n| Feature | Status | Notes |\n| --- | --- | --- |\n| Headings | ✅ | Multiple levels |\n| Lists | ✅ | Ordered + unordered |\n| Code | ✅ | Inline + block |\n\n## Section: Links and emphasis\n\nReference: [Alter docs](https://example.com)\n\nRemember: **clarity beats volume**. Write fewer lines but make each line carry weight.\n\n## Section: Code block\n\n```bash\n./run.sh --topic \"Guides\" --limit 5\n```\n\n## Closing\n\nIf any section renders oddly, note the heading and the element type so we can reproduce it quickly.",
			CreatedAt: now.Add(-6 * time.Hour).UnixMilli(),
		},
	}

	for _, item := range items {
		if err := SaveFeedItem(item); err != nil {
			panic(err)
		}
	}
}
