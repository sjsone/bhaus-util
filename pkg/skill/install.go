package skill

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// InstallSkill copies every file of skillToInstall into <targetDirectoryPath>/<skill name>.
//
// Existing files are overwritten
func InstallSkill(skillToInstall *Skill, targetDirectoryPath string) error {
	skillFolderPath := path.Join(targetDirectoryPath, skillToInstall.Name)

	for _, embeddedPath := range skillToInstall.Files {
		rel := strings.TrimPrefix(embeddedPath, skillToInstall.SkillFsPath+"/")
		destPath := filepath.Join(skillFolderPath, filepath.FromSlash(rel))

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		if err := copyEmbeddedFile(skillToInstall.fs, embeddedPath, destPath); err != nil {
			return err
		}
	}

	return nil
}

func copyEmbeddedFile(srcFS fs.FS, srcPath, destPath string) error {
	src, err := srcFS.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	// os.Create requests permission 0666
	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dest, src); err != nil {
		dest.Close()
		return err
	}

	// The function returns the close error instead of deferring it.
	// A close error here means the copy did not reach disk (for example on a full filesystem).
	return dest.Close()
}
