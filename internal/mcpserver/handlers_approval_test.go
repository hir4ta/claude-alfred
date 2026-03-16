package mcpserver

import (
	"strings"
	"testing"

	"github.com/hir4ta/claude-alfred/internal/spec"
)

func TestRequireApprovalGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		size         spec.SpecSize
		reviewStatus spec.ReviewStatus
		createReview bool // create a review JSON file with approved status
		wantErr      bool
		errContains  string
	}{
		{
			name:         "M_pending_rejects",
			size:         spec.SizeM,
			reviewStatus: spec.ReviewPending,
			wantErr:      true,
			errContains:  "requires review",
		},
		{
			name:         "L_approved_without_json_rejects",
			size:         spec.SizeL,
			reviewStatus: spec.ReviewApproved,
			wantErr:      true,
			errContains:  "no approved review file",
		},
		{
			name:         "L_approved_with_json_allows",
			size:         spec.SizeL,
			reviewStatus: spec.ReviewApproved,
			createReview: true,
		},
		{
			name:         "XL_changes_requested_rejects",
			size:         spec.SizeXL,
			reviewStatus: spec.ReviewChangesRequested,
			wantErr:      true,
			errContains:  "unresolved review comments",
		},
		{
			name: "S_exempt",
			size: spec.SizeS,
		},
		{
			name: "D_exempt",
			size: spec.SizeDelta,
		},
		{
			name:         "L_pending_rejects",
			size:         spec.SizeL,
			reviewStatus: "",
			wantErr:      true,
			errContains:  "requires review",
		},
		{
			name:         "M_empty_status_rejects",
			size:         spec.SizeM,
			reviewStatus: "",
			wantErr:      true,
			errContains:  "requires review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			slug := "test-task"

			// Create a spec to set up _active.md.
			opts := []spec.InitOption{}
			if tt.size != "" {
				opts = append(opts, spec.WithSize(tt.size))
			}
			_, err := spec.Init(dir, slug, "test description for approval gate", opts...)
			if err != nil {
				t.Fatalf("spec.Init: %v", err)
			}

			// Set review status if specified.
			if tt.reviewStatus != "" {
				if err := spec.SetReviewStatus(dir, slug, tt.reviewStatus); err != nil {
					t.Fatalf("SetReviewStatus: %v", err)
				}
			}

			// Create approved review JSON if requested.
			if tt.createReview {
				sd := &spec.SpecDir{ProjectPath: dir, TaskSlug: slug}
				review := &spec.Review{
					Status:  spec.ReviewApproved,
					Summary: "test approval",
				}
				if err := sd.SaveReview(review); err != nil {
					t.Fatalf("SaveReview: %v", err)
				}
			}

			err = requireApprovalGate(dir, slug)
			if tt.wantErr {
				if err == nil {
					t.Errorf("requireApprovalGate() = nil, want error containing %q", tt.errContains)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("requireApprovalGate() error = %q, want containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("requireApprovalGate() = %v, want nil", err)
				}
			}
		})
	}
}
