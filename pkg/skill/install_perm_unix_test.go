//go:build unix

package skill

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestInstallSkillRequestsNarrowDirectoryPerms checks the permission mode
// InstallSkill asks the OS for. It does not check the mode the OS grants.
//
// The test clears the umask first. Under the normal umask of 022, this check
// would be pointless. MkdirAll(os.ModePerm) and MkdirAll(0o755) both produce
// mode 0755 under that umask. A test based on the resulting mode would pass
// even if InstallSkill asked for the wrong, too-open mode (0777). Clearing the
// umask makes the two requests produce different results, so the test can
// tell them apart.
//
// Containers and CI images often run with a permissive umask. In that
// environment, a wrong permission request does real damage: it can leave
// files writable by any user or process on the system.
func TestInstallSkillRequestsNarrowDirectoryPerms(t *testing.T) {
	old := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(old) })

	dir := t.TempDir()
	if err := InstallSkill(find(t, "bhaus"), dir); err != nil {
		t.Fatalf("InstallSkill: %v", err)
	}

	for _, p := range []string{
		filepath.Join(dir, "bhaus"),
		filepath.Join(dir, "bhaus", "references"),
		filepath.Join(dir, "bhaus", "SKILL.md"),
		filepath.Join(dir, "bhaus", "references", "language-spec.md"),
	} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if perm := info.Mode().Perm(); perm&0o022 != 0 {
			t.Errorf("%s created with %#o; must not be group/other writable", p, perm)
		}
	}
}
