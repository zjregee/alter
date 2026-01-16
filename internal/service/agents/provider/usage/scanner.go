package usage

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/zjregee/alter/internal/utils"
)

var logger = utils.GetLogger()

const (
	ProviderCodex  = "codex"
	ProviderClaude = "claude"

	MaxCodexLineBytes  = 32 * 1024
	MaxClaudeLineBytes = 64 * 1024
)

type ScannerOptions struct {
	CodexSessionsRoot   string
	ClaudeProjectsRoots []string
	CacheRoot           string
	RefreshMinInterval  time.Duration
}

type DayRange struct {
	SinceKey     string
	UntilKey     string
	ScanSinceKey string
	ScanUntilKey string
}

func NewDayRange(since, until time.Time) DayRange {
	return DayRange{
		SinceKey:     dayKey(since),
		UntilKey:     dayKey(until),
		ScanSinceKey: dayKey(since.AddDate(0, 0, -1)),
		ScanUntilKey: dayKey(until.AddDate(0, 0, 1)),
	}
}

func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}

func isInRange(dayKey, since, until string) bool {
	return dayKey >= since && dayKey <= until
}

func parseDayKey(key string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", key, time.Local)
}

func LoadDailyReport(provider string, since, until, now time.Time, options ScannerOptions) (*DailyReport, error) {
	rng := NewDayRange(since, until)
	switch provider {
	case ProviderCodex:
		return loadCodexDaily(rng, now, options)
	case ProviderClaude:
		return loadClaudeDaily(rng, now, options)
	default:
		return &DailyReport{Data: []DailyReportEntry{}}, nil
	}
}

type providerHooks struct {
	name        string
	listFunc    func(rng DayRange, options ScannerOptions) ([]string, error)
	parseFunc   func(filePath string, rng DayRange) (*FileUsage, error)
	buildReport func(cache *Cache, rng DayRange) *DailyReport
}

func loadDailyForProvider(hooks providerHooks, rng DayRange, now time.Time, options ScannerOptions) (*DailyReport, error) {
	cache, err := LoadCache(hooks.name, options.CacheRoot)
	if err != nil {
		logger.Printf("[WARN] failed to load %s cache: %v", hooks.name, err)
		cache = NewCache()
	}

	nowMs := now.UnixMilli()
	shouldRefresh := options.RefreshMinInterval == 0 || cache.LastScanUnixMs == 0 || now.Sub(time.UnixMilli(cache.LastScanUnixMs)) > options.RefreshMinInterval

	if shouldRefresh {
		allFiles, err := hooks.listFunc(rng, options)
		if err != nil {
			return nil, err
		}
		filesInScan := make(map[string]struct{})
		for _, f := range allFiles {
			filesInScan[f] = struct{}{}
		}

		type parseResult struct {
			filePath string
			usage    *FileUsage
			err      error
		}

		var filesToParse []string
		for _, filePath := range allFiles {
			info, err := os.Stat(filePath)
			if err != nil {
				continue
			}
			mtimeMs := info.ModTime().UnixMilli()
			size := info.Size()

			if cached, ok := cache.Files[filePath]; ok && cached.MtimeUnixMs == mtimeMs && cached.Size == size {
				continue
			}
			filesToParse = append(filesToParse, filePath)
		}

		if len(filesToParse) > 0 {
			numWorkers := runtime.NumCPU()
			jobs := make(chan string, len(filesToParse))
			results := make(chan parseResult, len(filesToParse))
			var wg sync.WaitGroup

			for w := 0; w < numWorkers; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for filePath := range jobs {
						parsed, err := hooks.parseFunc(filePath, rng)
						results <- parseResult{filePath: filePath, usage: parsed, err: err}
					}
				}()
			}

			for _, filePath := range filesToParse {
				jobs <- filePath
			}
			close(jobs)

			wg.Wait()
			close(results)

			for result := range results {
				if result.err != nil {
					logger.Printf("[ERROR] error parsing %s file %s: %v", hooks.name, result.filePath, result.err)
					continue
				}

				if cached, ok := cache.Files[result.filePath]; ok {
					applyFileDays(cache, cached.Days, -1)
				}
				cache.Files[result.filePath] = *result.usage
				applyFileDays(cache, result.usage.Days, 1)
			}
		}

		for path := range cache.Files {
			if _, ok := filesInScan[path]; !ok {
				if old, ok := cache.Files[path]; ok {
					applyFileDays(cache, old.Days, -1)
				}
				delete(cache.Files, path)
			}
		}

		pruneDays(cache, rng.ScanSinceKey, rng.ScanUntilKey)
		cache.LastScanUnixMs = nowMs
		if err := SaveCache(hooks.name, cache, options.CacheRoot); err != nil {
			logger.Printf("[ERROR] failed to save %s cache: %v", hooks.name, err)
		}
	}

	return hooks.buildReport(cache, rng), nil
}

type costExtractor func(model string, packed []int) (inputTokens int, outputTokens int, cost *float64)

func buildReportFromCache(cache *Cache, rng DayRange, extractor costExtractor) *DailyReport {
	dayKeys := make([]string, 0, len(cache.Days))
	for k := range cache.Days {
		dayKeys = append(dayKeys, k)
	}
	sort.Strings(dayKeys)

	entries := make([]DailyReportEntry, 0, len(dayKeys))
	var totalInput, totalOutput, totalTokens int
	var totalCost float64
	var costSeen bool

	for _, day := range dayKeys {
		if !isInRange(day, rng.SinceKey, rng.UntilKey) {
			continue
		}
		models := cache.Days[day]
		modelNames := make([]string, 0, len(models))
		for k := range models {
			modelNames = append(modelNames, k)
		}
		sort.Strings(modelNames)

		var dayInput, dayOutput int
		breakdowns := make([]ModelBreakdown, 0, len(modelNames))
		var dayCost float64
		var dayCostSeen bool

		for _, model := range modelNames {
			packed := models[model]
			input, output, cost := extractor(model, packed)

			dayInput += input
			dayOutput += output

			breakdowns = append(breakdowns, ModelBreakdown{ModelName: model, CostUSD: cost})
			if cost != nil {
				dayCost += *cost
				dayCostSeen = true
			}
		}
		sort.Slice(breakdowns, func(i, j int) bool {
			if breakdowns[i].CostUSD == nil {
				return false
			}
			if breakdowns[j].CostUSD == nil {
				return true
			}
			return *breakdowns[i].CostUSD > *breakdowns[j].CostUSD
		})

		dayTotal := dayInput + dayOutput
		var entryCost *float64
		if dayCostSeen {
			entryCost = &dayCost
		}

		entries = append(entries, DailyReportEntry{
			Date:            day,
			InputTokens:     &dayInput,
			OutputTokens:    &dayOutput,
			TotalTokens:     &dayTotal,
			CostUSD:         entryCost,
			ModelsUsed:      modelNames,
			ModelBreakdowns: breakdowns,
		})

		totalInput += dayInput
		totalOutput += dayOutput
		totalTokens += dayTotal
		if entryCost != nil {
			totalCost += *entryCost
			costSeen = true
		}
	}

	var summary *DailyReportSummary
	if len(entries) > 0 {
		summary = &DailyReportSummary{
			TotalInputTokens:  &totalInput,
			TotalOutputTokens: &totalOutput,
			TotalTokens:       &totalTokens,
		}
		if costSeen {
			summary.TotalCostUSD = &totalCost
		}
	}

	return &DailyReport{Data: entries, Summary: summary}
}

func codexCostExtractor(model string, packed []int) (int, int, *float64) {
	if len(packed) < 3 {
		return 0, 0, nil
	}
	input, cached, output := packed[0], packed[1], packed[2]
	cost := GetCodexCostUSD(model, input, cached, output)
	return input, output, cost
}

func listCodexLogs(rng DayRange, options ScannerOptions) ([]string, error) {
	root, err := defaultCodexSessionsRoot(options)
	if err != nil {
		return nil, err
	}

	var files []string
	since, err := parseDayKey(rng.ScanSinceKey)
	if err != nil {
		return nil, err
	}
	until, err := parseDayKey(rng.ScanUntilKey)
	if err != nil {
		return nil, err
	}

	for d := since; !d.After(until); d = d.AddDate(0, 0, 1) {
		dayDir := filepath.Join(root, d.Format("2006"), d.Format("01"), d.Format("02"))
		entries, err := os.ReadDir(dayDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".jsonl") {
				files = append(files, filepath.Join(dayDir, entry.Name()))
			}
		}
	}
	return files, nil
}

func defaultCodexSessionsRoot(options ScannerOptions) (string, error) {
	if options.CodexSessionsRoot != "" {
		return options.CodexSessionsRoot, nil
	}
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		return filepath.Join(codexHome, "sessions"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "sessions"), nil
}

func parseCodexFile(filePath string, rng DayRange) (*FileUsage, error) {
	fileUsage := &FileUsage{Days: make(map[string]map[string][]int)}
	var currentModel string
	var previousTotals struct {
		input  int
		cached int
		output int
	}

	add := func(dayKey, model string, input, cached, output int) {
		if !isInRange(dayKey, rng.ScanSinceKey, rng.ScanUntilKey) {
			return
		}
		normModel := normalizeCodexModel(model)
		if _, ok := fileUsage.Days[dayKey]; !ok {
			fileUsage.Days[dayKey] = make(map[string][]int)
		}
		dayModels := fileUsage.Days[dayKey]
		packed := dayModels[normModel]
		if packed == nil {
			packed = []int{0, 0, 0}
		}
		packed[0] += input
		packed[1] += cached
		packed[2] += output
		dayModels[normModel] = packed
	}

	err := ScanJSONL(filePath, MaxCodexLineBytes, func(line Line) {
		if line.WasDiscarded || len(line.Bytes) == 0 {
			return
		}

		if !gjson.ValidBytes(line.Bytes) {
			return
		}

		res := gjson.ParseBytes(line.Bytes)
		lineType := res.Get("type").String()

		if lineType != "event_msg" && lineType != "turn_context" {
			return
		}
		if lineType == "event_msg" && res.Get("payload.type").String() != "token_count" {
			return
		}

		tsStr := res.Get("timestamp").String()
		ts, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return
		}
		dayKey := dayKey(ts)

		if lineType == "turn_context" {
			model := res.Get("payload.model").String()
			if model == "" {
				model = res.Get("payload.info.model").String()
			}
			if model != "" {
				currentModel = model
			}
			return
		}

		payload := res.Get("payload")
		model := payload.Get("info.model").String()
		if model == "" {
			model = payload.Get("info.model_name").String()
		}
		if model == "" {
			model = payload.Get("model").String()
		}
		if model == "" {
			model = res.Get("model").String()
		}
		if model == "" && currentModel != "" {
			model = currentModel
		}
		if model == "" {
			model = "unknown_codex_model"
		}

		toInt := func(r gjson.Result) int {
			return int(r.Int())
		}

		var deltaInput, deltaCached, deltaOutput int
		total := payload.Get("info.total_token_usage")
		last := payload.Get("info.last_token_usage")

		if total.Exists() {
			input := toInt(total.Get("input_tokens"))
			cached := toInt(total.Get("cached_input_tokens"))
			if cached == 0 {
				cached = toInt(total.Get("cache_read_input_tokens"))
			}
			output := toInt(total.Get("output_tokens"))

			deltaInput = max(0, input-previousTotals.input)
			deltaCached = max(0, cached-previousTotals.cached)
			deltaOutput = max(0, output-previousTotals.output)
			previousTotals.input = input
			previousTotals.cached = cached
			previousTotals.output = output
		} else if last.Exists() {
			deltaInput = max(0, toInt(last.Get("input_tokens")))
			deltaCached = max(0, toInt(last.Get("cached_input_tokens")))
			if deltaCached == 0 {
				deltaCached = toInt(last.Get("cache_read_input_tokens"))
			}
			deltaOutput = max(0, toInt(last.Get("output_tokens")))
			previousTotals.input += deltaInput
			previousTotals.cached += deltaCached
			previousTotals.output += deltaOutput
		} else {
			return
		}

		if deltaInput == 0 && deltaCached == 0 && deltaOutput == 0 {
			return
		}

		add(dayKey, model, deltaInput, min(deltaCached, deltaInput), deltaOutput)
	})

	if err != nil {
		logger.Printf("[ERROR] error scanning %s: %v", filePath, err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	fileUsage.MtimeUnixMs = info.ModTime().UnixMilli()
	fileUsage.Size = info.Size()

	return fileUsage, nil
}

func loadCodexDaily(rng DayRange, now time.Time, options ScannerOptions) (*DailyReport, error) {
	hooks := providerHooks{
		name:      ProviderCodex,
		listFunc:  listCodexLogs,
		parseFunc: parseCodexFile,
		buildReport: func(cache *Cache, rng DayRange) *DailyReport {
			return buildReportFromCache(cache, rng, codexCostExtractor)
		},
	}
	return loadDailyForProvider(hooks, rng, now, options)
}

func claudeCostExtractor(model string, packed []int) (int, int, *float64) {
	if len(packed) < 4 {
		return 0, 0, nil
	}
	input, cacheRead, cacheCreate, output := packed[0], packed[1], packed[2], packed[3]
	inputTotal := input + cacheRead + cacheCreate

	cost := GetClaudeCostUSD(model, input, cacheRead, cacheCreate, output)
	return inputTotal, output, cost
}

func listClaudeLogs(rng DayRange, options ScannerOptions) ([]string, error) {
	roots := defaultClaudeProjectsRoots(options)
	var files []string
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if info.Size() <= 0 {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			logger.Printf("[ERROR] error walking claude project root %s: %v", root, err)
		}
	}
	return files, nil
}

func defaultClaudeProjectsRoots(options ScannerOptions) []string {
	if len(options.ClaudeProjectsRoots) > 0 {
		return options.ClaudeProjectsRoots
	}

	var roots []string
	if claudeConfigDir := os.Getenv("CLAUDE_CONFIG_DIR"); claudeConfigDir != "" {
		for _, part := range strings.Split(claudeConfigDir, ",") {
			raw := strings.TrimSpace(part)
			if raw == "" {
				continue
			}
			if strings.HasSuffix(raw, "projects") {
				roots = append(roots, raw)
			} else {
				roots = append(roots, filepath.Join(raw, "projects"))
			}
		}
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			roots = append(roots, filepath.Join(home, ".config", "claude", "projects"))
			roots = append(roots, filepath.Join(home, ".claude", "projects"))
		}
	}
	return roots
}

func parseClaudeFile(filePath string, rng DayRange) (*FileUsage, error) {
	fileUsage := &FileUsage{Days: make(map[string]map[string][]int)}

	add := func(dayKey, model string, tokens map[string]int) {
		if !isInRange(dayKey, rng.ScanSinceKey, rng.ScanUntilKey) {
			return
		}
		normModel := normalizeClaudeModel(model)
		if _, ok := fileUsage.Days[dayKey]; !ok {
			fileUsage.Days[dayKey] = make(map[string][]int)
		}
		dayModels := fileUsage.Days[dayKey]
		packed := dayModels[normModel]
		if packed == nil {
			packed = make([]int, 4)
		}
		packed[0] += tokens["input"]
		packed[1] += tokens["cacheRead"]
		packed[2] += tokens["cacheCreate"]
		packed[3] += tokens["output"]

		dayModels[normModel] = packed
	}

	err := ScanJSONL(filePath, MaxClaudeLineBytes, func(line Line) {
		if line.WasDiscarded || len(line.Bytes) == 0 {
			return
		}
		if !gjson.ValidBytes(line.Bytes) {
			return
		}
		res := gjson.ParseBytes(line.Bytes)
		if res.Get("type").String() != "assistant" || !res.Get("message.usage").Exists() {
			return
		}

		tsStr := res.Get("timestamp").String()
		ts, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return
		}
		dayKey := dayKey(ts)

		message := res.Get("message")
		model := message.Get("model").String()
		usage := message.Get("usage")

		toInt := func(r gjson.Result) int { return int(r.Int()) }

		tokens := map[string]int{
			"input":       toInt(usage.Get("input_tokens")),
			"cacheCreate": toInt(usage.Get("cache_creation_input_tokens")),
			"cacheRead":   toInt(usage.Get("cache_read_input_tokens")),
			"output":      toInt(usage.Get("output_tokens")),
		}
		if tokens["input"] == 0 && tokens["cacheCreate"] == 0 && tokens["cacheRead"] == 0 && tokens["output"] == 0 {
			return
		}

		add(dayKey, model, tokens)
	})

	if err != nil {
		logger.Printf("[ERROR] error scanning %s: %v", filePath, err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	fileUsage.MtimeUnixMs = info.ModTime().UnixMilli()
	fileUsage.Size = info.Size()

	return fileUsage, nil
}

func loadClaudeDaily(rng DayRange, now time.Time, options ScannerOptions) (*DailyReport, error) {
	hooks := providerHooks{
		name:      ProviderClaude,
		listFunc:  listClaudeLogs,
		parseFunc: parseClaudeFile,
		buildReport: func(cache *Cache, rng DayRange) *DailyReport {
			return buildReportFromCache(cache, rng, claudeCostExtractor)
		},
	}
	return loadDailyForProvider(hooks, rng, now, options)
}

func applyFileDays(cache *Cache, fileDays map[string]map[string][]int, sign int) {
	for day, models := range fileDays {
		if _, ok := cache.Days[day]; !ok {
			cache.Days[day] = make(map[string][]int)
		}
		dayModels := cache.Days[day]
		for model, packed := range models {
			existing := dayModels[model]
			merged := addPacked(existing, packed, sign)
			allZero := true
			for _, v := range merged {
				if v != 0 {
					allZero = false
					break
				}
			}
			if allZero {
				delete(dayModels, model)
			} else {
				dayModels[model] = merged
			}
		}
		if len(dayModels) == 0 {
			delete(cache.Days, day)
		}
	}
}

func pruneDays(cache *Cache, sinceKey, untilKey string) {
	for key := range cache.Days {
		if !isInRange(key, sinceKey, untilKey) {
			delete(cache.Days, key)
		}
	}
}

func addPacked(a, b []int, sign int) []int {
	lenA, lenB := len(a), len(b)
	maxLen := max(lenA, lenB)
	out := make([]int, maxLen)

	for i := 0; i < maxLen; i += 1 {
		valA := 0
		if i < lenA {
			valA = a[i]
		}
		valB := 0
		if i < lenB {
			valB = b[i]
		}
		out[i] = valA + sign*valB
	}
	return out
}
