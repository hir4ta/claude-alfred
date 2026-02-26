package hookhandler

import (
	"strings"

	"github.com/hir4ta/claude-buddy/internal/sessiondb"
	"github.com/hir4ta/claude-buddy/internal/store"
)

// DeepIntent represents a 4-layer understanding of the user's current task.
type DeepIntent struct {
	TaskType      TaskType // Level 1: bugfix/feature/refactor/test/explore/debug/review/docs
	Domain        string   // Level 2: auth/database/ui/api/config/infra/general
	WorkflowPhase Phase    // Level 3: reused from phases.go
	RiskProfile   string   // Level 4: conservative/balanced/aggressive
	Confidence    float64  // overall confidence [0, 1]
}

// AnalyzeDeepIntent builds a 4-layer intent model from the user prompt and session state.
func AnalyzeDeepIntent(sdb *sessiondb.SessionDB, prompt string, taskType TaskType) *DeepIntent {
	di := &DeepIntent{
		TaskType:   taskType,
		Domain:     detectDomain(prompt),
		RiskProfile: inferRiskProfile(sdb),
		Confidence: 0.5,
	}

	// Level 3: reuse existing phase detection.
	if progress := GetPhaseProgress(sdb); progress != nil {
		di.WorkflowPhase = progress.CurrentPhase
	}

	// Confidence: higher when more layers are populated.
	populated := 0
	if di.TaskType != TaskUnknown {
		populated++
	}
	if di.Domain != "general" {
		populated++
	}
	if di.WorkflowPhase != PhaseUnknown {
		populated++
	}
	if di.RiskProfile != "" {
		populated++
	}
	di.Confidence = float64(populated) / 4.0

	return di
}

// domainKeywords maps domain names to detection keywords.
var domainKeywords = map[string][]string{
	"auth":     {"auth", "login", "logout", "password", "token", "jwt", "oauth", "session", "credential", "認証", "ログイン"},
	"database": {"database", "db", "sql", "query", "migration", "schema", "table", "index", "postgres", "sqlite", "mysql", "データベース"},
	"ui":       {"ui", "component", "button", "form", "modal", "layout", "css", "style", "render", "display", "画面", "表示"},
	"api":      {"api", "endpoint", "handler", "route", "request", "response", "rest", "grpc", "middleware", "エンドポイント"},
	"config":   {"config", "setting", "env", "environment", "yaml", "toml", "json config", "設定", "環境"},
	"infra":    {"deploy", "docker", "ci", "cd", "pipeline", "kubernetes", "k8s", "terraform", "nginx", "デプロイ", "インフラ"},
	"test":     {"test", "spec", "mock", "stub", "fixture", "assertion", "coverage", "テスト", "カバレッジ"},
}

// detectDomain classifies the task domain from the user prompt using keyword matching.
func detectDomain(prompt string) string {
	lower := strings.ToLower(prompt)

	bestDomain := "general"
	bestScore := 0

	for domain, keywords := range domainKeywords {
		score := 0
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestDomain = domain
		}
	}

	return bestDomain
}

// inferRiskProfile determines the user's risk profile from behavioral data.
// conservative: many reads before writes, high test frequency
// aggressive: few reads, low test frequency, fast velocity
// balanced: everything else
func inferRiskProfile(sdb *sessiondb.SessionDB) string {
	st, err := store.OpenDefault()
	if err != nil {
		return "balanced"
	}
	defer st.Close()

	readWriteRatio, rwCount, _ := st.GetUserProfile("read_write_ratio")
	testFreq, tfCount, _ := st.GetUserProfile("test_frequency")

	// Need sufficient data to classify.
	if rwCount < 3 && tfCount < 3 {
		return "balanced"
	}

	conservative := 0
	aggressive := 0

	if rwCount >= 3 {
		if readWriteRatio > 3.0 {
			conservative++
		} else if readWriteRatio < 1.0 {
			aggressive++
		}
	}

	if tfCount >= 3 {
		if testFreq > 0.5 {
			conservative++
		} else if testFreq < 0.15 {
			aggressive++
		}
	}

	// Velocity can also indicate risk appetite.
	vel := getFloat(sdb, "ewma_tool_velocity")
	if vel > 10.0 {
		aggressive++
	} else if vel < 3.0 && vel > 0 {
		conservative++
	}

	switch {
	case conservative >= 2:
		return "conservative"
	case aggressive >= 2:
		return "aggressive"
	default:
		return "balanced"
	}
}
