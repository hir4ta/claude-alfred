package hookhandler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
	"github.com/hir4ta/claude-buddy/internal/store"
)

type sessionEndInput struct {
	CommonInput
	Reason string `json:"reason"`
}

func handleSessionEnd(input []byte) (*HookOutput, error) {
	var in sessionEndInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}

	sdb, err := sessiondb.Open(in.SessionID)
	if err != nil {
		return nil, nil
	}

	// Persist workflow sequence, session metrics, tool sequences, and user profile
	// before destroying session DB.
	persistWorkflowSequence(sdb, in.SessionID)
	persistSessionMetrics(sdb)
	persistUserProfile(sdb)
	persistCoChanges(sdb)
	mergeToolSequencesToStore(sdb)

	_ = sdb.Destroy()
	return nil, nil
}

// persistWorkflowSequence extracts the phase sequence from session_phases
// and saves it to the persistent store for future workflow learning.
func persistWorkflowSequence(sdb *sessiondb.SessionDB, sessionID string) {
	phases, err := sdb.GetPhaseSequence()
	if err != nil || len(phases) < 2 {
		return // not enough data to learn from
	}

	taskTypeStr, _ := sdb.GetContext("task_type")
	if taskTypeStr == "" {
		return
	}

	phaseCount, _ := sdb.PhaseCount()

	st, err := store.OpenDefault()
	if err != nil {
		return
	}
	defer st.Close()

	// Heuristic success: no recent unresolved failures.
	success := true
	failures, _ := sdb.RecentFailures(3)
	for _, f := range failures {
		if f.FilePath == "" {
			continue
		}
		unresolved, _, _ := sdb.HasUnresolvedFailure(f.FilePath)
		if unresolved {
			success = false
			break
		}
	}

	_ = st.InsertWorkflowSequence(sessionID, taskTypeStr, phases, success, phaseCount, 0)
}

// persistSessionMetrics extracts per-session metrics and feeds them into
// the adaptive baselines (Welford online algorithm) in the persistent store.
func persistSessionMetrics(sdb *sessiondb.SessionDB) {
	st, err := store.OpenDefault()
	if err != nil {
		return
	}
	defer st.Close()

	// Retry loop: max consecutive same-tool runs.
	events, err := sdb.RecentEvents(50)
	if err == nil && len(events) > 0 {
		maxConsecutive := 1
		consecutive := 1
		for i := 1; i < len(events); i++ {
			if events[i].ToolName == events[i-1].ToolName && events[i].InputHash == events[i-1].InputHash {
				consecutive++
				if consecutive > maxConsecutive {
					maxConsecutive = consecutive
				}
			} else {
				consecutive = 1
			}
		}
		_ = st.UpdateBaseline("retry_loop_consecutive", float64(maxConsecutive))

		// File hotspot: max writes to single file.
		fileWrites := make(map[uint64]int)
		maxWrites := 0
		for _, ev := range events {
			if ev.IsWrite {
				fileWrites[ev.InputHash]++
				if fileWrites[ev.InputHash] > maxWrites {
					maxWrites = fileWrites[ev.InputHash]
				}
			}
		}
		_ = st.UpdateBaseline("file_hotspot_writes", float64(maxWrites))

		// Distinct files modified.
		distinctFiles := make(map[uint64]bool)
		for _, ev := range events {
			if ev.IsWrite {
				distinctFiles[ev.InputHash] = true
			}
		}
		_ = st.UpdateBaseline("plan_mode_files", float64(len(distinctFiles)))
	}

	// No-progress metrics: tools in burst + elapsed minutes since burst start.
	tc, _, _, _ := sdb.BurstState()
	_ = st.UpdateBaseline("no_progress_tools", float64(tc))

	if startTime, err := sdb.BurstStartTime(); err == nil && !startTime.IsZero() {
		elapsed := time.Since(startTime).Minutes()
		_ = st.UpdateBaseline("no_progress_minutes", elapsed)
	}

	// Compaction burst: record burst tool count at session end as proxy for
	// typical burst size when compaction risk is evaluated.
	compacts, _ := sdb.CompactsInWindow(60)
	if compacts > 0 {
		_ = st.UpdateBaseline("compaction_burst_tools", float64(tc))
	}

	// EWMA error rate.
	errRate := getFloat(sdb, "ewma_error_rate")
	_ = st.UpdateBaseline("debug_error_rate", errRate)

	// Phase distribution metrics.
	phases, err := sdb.GetRawPhaseSequence(20)
	if err == nil && len(phases) > 0 {
		dist := phaseDist(phases)
		_ = st.UpdateBaseline("explore_read_pct", dist["read"])

		editBashCount := countTransitions(phases, "write", "compile") +
			countTransitions(phases, "write", "test")
		_ = st.UpdateBaseline("debug_edit_cycles", float64(editBashCount))
	}
}

// persistUserProfile extracts session-level coding style metrics
// and updates the persistent user profile with EWMA smoothing.
func persistUserProfile(sdb *sessiondb.SessionDB) {
	st, err := store.OpenDefault()
	if err != nil {
		return
	}
	defer st.Close()

	tc, hasWrite, _, _ := sdb.BurstState()
	if tc == 0 {
		return
	}

	// tools_per_burst: total tools in this session burst.
	_ = st.UpdateUserProfile("tools_per_burst", float64(tc))

	// read_write_ratio: proportion of read vs write tools.
	events, err := sdb.RecentEvents(50)
	if err == nil && len(events) > 0 {
		reads, writes := 0, 0
		for _, ev := range events {
			if ev.IsWrite {
				writes++
			} else {
				reads++
			}
		}
		if writes > 0 {
			_ = st.UpdateUserProfile("read_write_ratio", float64(reads)/float64(writes))
		}
	}

	// test_frequency: did the user run tests? (1.0 = yes, 0.0 = no)
	hasTestRun, _ := sdb.GetContext("has_test_run")
	if hasTestRun == "true" {
		_ = st.UpdateUserProfile("test_frequency", 1.0)
	} else if hasWrite {
		_ = st.UpdateUserProfile("test_frequency", 0.0)
	}

	// compact_frequency: number of compactions in this session.
	compacts, _ := sdb.CompactsInWindow(525600) // 1 year — effectively all time in session
	_ = st.UpdateUserProfile("compact_frequency", float64(compacts))

	// avg_session_duration: session length in minutes.
	if startTime, err := sdb.BurstStartTime(); err == nil && !startTime.IsZero() {
		duration := time.Since(startTime).Minutes()
		_ = st.UpdateUserProfile("avg_session_duration", duration)
	}

	// reads_before_write: average number of read tools before the first write.
	if err == nil && len(events) > 0 {
		readsBeforeFirstWrite := 0
		foundWrite := false
		// Events are newest-first; iterate in reverse for chronological order.
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].IsWrite {
				foundWrite = true
				break
			}
			readsBeforeFirstWrite++
		}
		if foundWrite {
			_ = st.UpdateUserProfile("reads_before_write", float64(readsBeforeFirstWrite))
		}
	}

	// test_before_edit: did user run tests before editing? (1.0=yes, 0.0=no)
	if err == nil && len(events) > 0 {
		sawTest := false
		sawWrite := false
		for i := len(events) - 1; i >= 0; i-- {
			name := events[i].ToolName
			if name == "Bash" && !events[i].IsWrite {
				// Approximate: Bash non-write is often a test run.
				sawTest = true
			}
			if events[i].IsWrite && !sawWrite {
				sawWrite = true
				if sawTest {
					_ = st.UpdateUserProfile("test_before_edit", 1.0)
				} else {
					_ = st.UpdateUserProfile("test_before_edit", 0.0)
				}
			}
		}
	}

	// suggestion_follow_rate: fraction of helpful feedback (from feedbacks table).
	if stats, ferr := st.AllFeedbackStats(); ferr == nil && stats.TotalCount > 0 {
		followRate := float64(stats.Helpful) / float64(stats.TotalCount)
		_ = st.UpdateUserProfile("suggestion_follow_rate", followRate)
	}
}

// persistCoChanges records file co-change pairs from the working set.
func persistCoChanges(sdb *sessiondb.SessionDB) {
	files, err := sdb.GetWorkingSetFiles()
	if err != nil || len(files) < 2 {
		return
	}

	st, err := store.OpenDefault()
	if err != nil {
		return
	}
	defer st.Close()

	_ = st.RecordCoChanges(files)
}

// mergeToolSequencesToStore merges session-local tool bigrams and trigrams
// into the global persistent store for cross-session prediction.
func mergeToolSequencesToStore(sdb *sessiondb.SessionDB) {
	st, err := store.OpenDefault()
	if err != nil {
		return
	}
	defer st.Close()

	bigrams, err := sdb.AllToolSequences()
	if err == nil && len(bigrams) > 0 {
		_ = st.MergeToolSequences(bigrams)
	}

	trigrams, err := sdb.AllToolTrigrams()
	if err == nil && len(trigrams) > 0 {
		_ = st.MergeToolTrigrams(trigrams)
	}
}
