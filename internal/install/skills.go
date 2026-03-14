package install

import (
	"embed"
	"io/fs"
	"path"
)

//go:embed content/skills/*/SKILL.md content/skills/*/*.md content/skills/*/checklists/*.md
var skillsFS embed.FS

type skillDef struct {
	Dir     string // directory name under skills/
	Content string // SKILL.md content
}

type skillFileDef struct {
	Dir  string // directory name under skills/
	File string // filename (e.g., "best-practices.md")
	Data string // file content
}

// loadSkills reads all skill definitions from the embedded filesystem.
func loadSkills() []skillDef {
	var skills []skillDef
	entries, err := fs.ReadDir(skillsFS, "content/skills")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(skillsFS, path.Join("content/skills", e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		skills = append(skills, skillDef{Dir: e.Name(), Content: string(data)})
	}
	return skills
}

// loadSkillSupportFiles reads all non-SKILL.md files from skill directories,
// including files in subdirectories (e.g., checklists/*.md).
func loadSkillSupportFiles() []skillFileDef {
	var files []skillFileDef
	entries, err := fs.ReadDir(skillsFS, "content/skills")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillDir := path.Join("content/skills", e.Name())
		collectFiles(skillsFS, skillDir, e.Name(), "", &files)
	}
	return files
}

// collectFiles recursively reads files from a skill directory.
func collectFiles(fsys fs.FS, dir, skillDir, prefix string, files *[]skillFileDef) {
	dirEntries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return
	}
	for _, f := range dirEntries {
		relName := path.Join(prefix, f.Name())
		if f.IsDir() {
			collectFiles(fsys, path.Join(dir, f.Name()), skillDir, relName, files)
			continue
		}
		if f.Name() == "SKILL.md" && prefix == "" {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		*files = append(*files, skillFileDef{Dir: skillDir, File: relName, Data: string(data)})
	}
}

// deprecatedSkillDirs lists skill directories from previous versions that
// should be cleaned up during install/uninstall.
var deprecatedSkillDirs = []string{
	// v0.1-v0.19 era
	"init",
	"alfred-unstuck",
	"alfred-checkpoint",
	"alfred-before-commit",
	"alfred-impact",
	"alfred-review",
	"alfred-estimate",
	"alfred-error-recovery",
	"alfred-test-guidance",
	"alfred-predict",
	// v0.20-v0.22 era
	"alfred-recover",
	"alfred-gate",
	"alfred-analyze",
	"alfred-forecast",
	"alfred-context-recovery",
	"alfred-crawl",
	// v0.23 era (alfred- prefix removed in v0.24)
	"alfred-create-skill",
	"alfred-create-rule",
	"alfred-create-hook",
	"alfred-create-agent",
	"alfred-create-mcp",
	"alfred-create-claude-md",
	"alfred-create-memory",
	"alfred-review",
	"alfred-audit",
	"alfred-learn",
	"alfred-preferences",
	"alfred-update-docs",
	"alfred-update",
	"alfred-setup",
	"alfred-migrate",
	"alfred-explain",
	// v0.24-v0.26 era (renamed to alfred-style in v0.27)
	"create-skill",
	"create-rule",
	"create-hook",
	"create-agent",
	"create-mcp",
	"create-claude-md",
	"create-memory",
	"review",
	"audit",
	"learn",
	"preferences",
	"update-docs",
	"update",
	"setup",
	"migrate",
	"explain",
	// v0.27-v0.28 era (consolidated into configure/setup)
	"inspect",
	"harvest",
	"prepare",
	"polish",
	"greetings",
	"brief",
	"memorize",
}
