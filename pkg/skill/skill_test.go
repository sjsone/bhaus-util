package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func find(t *testing.T, name string) *Skill {
	t.Helper()
	for _, s := range GetAvailableSkills() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("skill %q is not bundled; available: %v", name, names())
	return nil
}

func names() []string {
	var out []string
	for _, s := range GetAvailableSkills() {
		out = append(out, s.Name)
	}
	return out
}

func TestGetAvailableSkillsBundlesBhaus(t *testing.T) {
	s := find(t, "bhaus")

	if s.SkillFsPath == "" {
		t.Error("SkillFsPath is empty")
	}
	if len(s.Files) == 0 {
		t.Fatal("skill has no files")
	}

	// The embedded tree is nested. At minimum, SKILL.md and a references/ entry must be present.
	var hasManifest, hasNested bool
	for _, f := range s.Files {
		if filepath.Base(f) == "SKILL.md" {
			hasManifest = true
		}
		if filepath.Base(filepath.Dir(f)) == "references" {
			hasNested = true
		}
	}
	if !hasManifest {
		t.Errorf("no SKILL.md among %v", s.Files)
	}
	if !hasNested {
		t.Errorf("no references/ entry among %v", s.Files)
	}
}

// TestGetAvailableSkillsBundlesBhausImport guards the import-direction skill
// (source code -> .bhaus), which ships alongside the bhaus skill.
func TestGetAvailableSkillsBundlesBhausImport(t *testing.T) {
	s := find(t, "bhaus-import")

	if s.SkillFsPath == "" {
		t.Error("SkillFsPath is empty")
	}
	if len(s.Files) == 0 {
		t.Fatal("skill has no files")
	}

	var hasManifest, hasLanguageRefs int
	for _, f := range s.Files {
		switch filepath.Base(f) {
		case "SKILL.md":
			hasManifest++
		case "go.md", "php.md", "swift.md", "language-spec.md":
			hasLanguageRefs++
		}
	}
	if hasManifest != 1 {
		t.Errorf("expected exactly one SKILL.md among %v", s.Files)
	}
	if hasLanguageRefs != 4 {
		t.Errorf("expected go/php/swift/language-spec references among %v", s.Files)
	}
}

// TestInstallSkillPreservesLayout is the guard for the "copy the whole skill
// folder, not just the file" behavior: nested paths must survive the copy.
func TestInstallSkillPreservesLayout(t *testing.T) {
	dir := t.TempDir()
	s := find(t, "bhaus")

	if err := InstallSkill(s, dir); err != nil {
		t.Fatalf("InstallSkill: %v", err)
	}

	root := filepath.Join(dir, "bhaus")
	manifest := filepath.Join(root, "SKILL.md")
	info, err := os.Stat(manifest)
	if err != nil {
		t.Fatalf("expected %s: %v", manifest, err)
	}
	if info.Size() == 0 {
		t.Error("SKILL.md was created empty")
	}

	spec := filepath.Join(root, "references", "language-spec.md")
	if _, err := os.Stat(spec); err != nil {
		t.Errorf("nested file not copied: %v", err)
	}
}

func TestInstallSkillOverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	s := find(t, "bhaus")

	if err := InstallSkill(s, dir); err != nil {
		t.Fatalf("first install: %v", err)
	}

	manifest := filepath.Join(dir, "bhaus", "SKILL.md")
	if err := os.WriteFile(manifest, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallSkill(s, dir); err != nil {
		t.Fatalf("second install: %v", err)
	}

	got, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) == "stale" {
		t.Error("existing file was not overwritten")
	}
}

// TestInstallSkillReportsUncreatableTarget proves the error is returned rather
// than swallowed: nothing can be created underneath a regular file.
func TestInstallSkillReportsUncreatableTarget(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallSkill(find(t, "bhaus"), filepath.Join(blocker, "sub")); err == nil {
		t.Fatal("InstallSkill returned nil for an uncreatable target")
	}
}
