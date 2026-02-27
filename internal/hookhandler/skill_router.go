package hookhandler

// SkillHintForPattern returns a skill invocation instruction for the given
// delivery pattern, or "" if no skill mapping exists. The instruction is
// designed to be appended to nudge messages so Claude invokes the skill
// automatically rather than retrying the failing approach.
func SkillHintForPattern(pattern string) string {
	switch pattern {
	case "retry-loop", "code-quality":
		return skillHint("claude-buddy:buddy-recover",
			"break out of this failure loop")
	case "test-correlation":
		return skillHint("claude-buddy:buddy-recover",
			"get targeted debugging strategy for failing tests")
	case "stale-read":
		return "" // simple re-read, no skill needed
	case "past-solution", "file-knowledge":
		return skillHint("claude-buddy:buddy-recover",
			"retrieve past resolution diffs for this error")
	case "workflow":
		return skillHint("claude-buddy:buddy-gate",
			"check session health and workflow alignment")
	case "strategic", "playbook":
		return "" // coaching-level, not actionable via skill
	default:
		return ""
	}
}

// SkillHintForEpisode returns a skill invocation instruction for the given
// episode detection. These are stronger than pattern hints because episodes
// indicate an active failure spiral that needs immediate intervention.
func SkillHintForEpisode(episodeName string) string {
	switch episodeName {
	case "retry_cascade", "edit_fail_spiral":
		return skillDenyHint("claude-buddy:buddy-recover",
			"analyze this failure before retrying")
	case "test_fail_fixup":
		return skillDenyHint("claude-buddy:buddy-recover",
			"analyze test failures before another fix attempt")
	case "explore_to_stuck":
		return skillHint("claude-buddy:buddy-gate",
			"assess progress and consider narrowing scope")
	case "context_overload":
		return skillDenyHint("claude-buddy:buddy-context-recovery",
			"preserve working context before the next compaction")
	case "learned_episode", "trajectory_match":
		return skillHint("claude-buddy:buddy-recover",
			"check past session knowledge for this failure pattern")
	default:
		return ""
	}
}

// SkillHintForPhase returns a skill invocation instruction appropriate for
// the given workflow phase transition. Used by coaching/playbook generation.
func SkillHintForPhase(phase string) string {
	switch phase {
	case "explore", "read":
		return skillHint("claude-buddy:buddy-forecast",
			"estimate task complexity before diving in")
	case "implement", "write":
		return skillHint("claude-buddy:buddy-analyze",
			"analyze blast radius of planned changes")
	case "test", "verify", "compile":
		return skillHint("claude-buddy:buddy-gate",
			"verify code quality before running tests")
	default:
		return ""
	}
}

// skillHint formats a standard skill invocation instruction.
func skillHint(skill, purpose string) string {
	return "\nIMPORTANT: Use the Skill tool now: skill=\"" + skill + "\" to " + purpose + "."
}

// skillDenyHint formats a stronger instruction used in deny/block contexts.
func skillDenyHint(skill, purpose string) string {
	return "\n→ STOP: Use the Skill tool: skill=\"" + skill + "\" to " + purpose +
		".\n→ Do NOT retry the same operation."
}
