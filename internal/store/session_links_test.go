package store

import (
	"context"
	"testing"
)

func TestSessionLinks(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	t.Run("link and resolve", func(t *testing.T) {
		err := st.LinkSession(ctx, &SessionLink{
			ClaudeSessionID: "session-2",
			MasterSessionID: "session-1",
			ProjectPath:     "/project",
			TaskSlug:        "my-task",
		})
		if err != nil {
			t.Fatalf("LinkSession: %v", err)
		}

		master := st.ResolveMasterSession(ctx, "session-2")
		if master != "session-1" {
			t.Errorf("ResolveMasterSession(session-2) = %q, want session-1", master)
		}
	})

	t.Run("resolve unlinked returns self", func(t *testing.T) {
		master := st.ResolveMasterSession(ctx, "unknown-session")
		if master != "unknown-session" {
			t.Errorf("ResolveMasterSession(unknown) = %q, want unknown-session", master)
		}
	})

	t.Run("idempotent link", func(t *testing.T) {
		err := st.LinkSession(ctx, &SessionLink{
			ClaudeSessionID: "session-2",
			MasterSessionID: "session-1",
			ProjectPath:     "/project",
			TaskSlug:        "my-task",
		})
		if err != nil {
			t.Fatalf("second LinkSession should be idempotent: %v", err)
		}
	})

	t.Run("continuity", func(t *testing.T) {
		// Add another linked session.
		st.LinkSession(ctx, &SessionLink{
			ClaudeSessionID: "session-3",
			MasterSessionID: "session-1",
			ProjectPath:     "/project",
			TaskSlug:        "my-task",
		})

		sc, err := st.GetSessionContinuity(ctx, "session-1")
		if err != nil {
			t.Fatalf("GetSessionContinuity: %v", err)
		}
		if sc.CompactCount != 2 {
			t.Errorf("CompactCount = %d, want 2", sc.CompactCount)
		}
		if len(sc.LinkedSessions) != 2 {
			t.Errorf("LinkedSessions = %v, want 2 entries", sc.LinkedSessions)
		}
	})

	t.Run("chain resolution direct", func(t *testing.T) {
		// session-3 links to session-1 (master).
		master := st.ResolveMasterSession(ctx, "session-3")
		if master != "session-1" {
			t.Errorf("chain resolution: got %q, want session-1", master)
		}
	})

	t.Run("transitive chain resolution", func(t *testing.T) {
		// Create a 3-hop chain: session-6 → session-5 → session-4
		st.LinkSession(ctx, &SessionLink{
			ClaudeSessionID: "session-5",
			MasterSessionID: "session-4",
			ProjectPath:     "/project",
			TaskSlug:        "chain-test",
		})
		st.LinkSession(ctx, &SessionLink{
			ClaudeSessionID: "session-6",
			MasterSessionID: "session-5",
			ProjectPath:     "/project",
			TaskSlug:        "chain-test",
		})

		// Resolving session-6 should walk session-6→session-5→session-4.
		master := st.ResolveMasterSession(ctx, "session-6")
		if master != "session-4" {
			t.Errorf("transitive chain: got %q, want session-4", master)
		}
	})
}

func TestDocRowSubType(t *testing.T) {
	t.Parallel()
	st := openTestStore(t)
	ctx := context.Background()

	// Save a decision memory.
	id, changed, err := st.UpsertDoc(ctx, &DocRow{
		URL:         "memory://test/decision-1",
		SectionPath: "test > decision > use FTS5",
		Content:     "Decided to use FTS5 for full-text search",
		SourceType:  SourceMemory,
		SubType:     SubTypeDecision,
		TTLDays:     0,
	})
	if err != nil {
		t.Fatalf("UpsertDoc: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for new doc")
	}

	// Verify sub_type is persisted.
	docs, err := st.GetDocsByIDs(ctx, []int64{id})
	if err != nil {
		t.Fatalf("GetDocsByIDs: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("GetDocsByIDs returned %d docs, want 1", len(docs))
	}
	if docs[0].SubType != SubTypeDecision {
		t.Errorf("SubType = %q, want %q", docs[0].SubType, SubTypeDecision)
	}

	// Save a general memory (default sub_type).
	_, _, err = st.UpsertDoc(ctx, &DocRow{
		URL:         "memory://test/general-1",
		SectionPath: "test > general > note",
		Content:     "Just a general note",
		SourceType:  SourceMemory,
		TTLDays:     0,
	})
	if err != nil {
		t.Fatalf("UpsertDoc general: %v", err)
	}

	// Verify default sub_type.
	results, err := st.SearchDocsByURLPrefix(ctx, "memory://test/general", 10)
	if err != nil {
		t.Fatalf("SearchDocsByURLPrefix: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].SubType != SubTypeGeneral {
		t.Errorf("default SubType = %q, want %q", results[0].SubType, SubTypeGeneral)
	}
}

