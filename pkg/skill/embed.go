package skill

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
)

//go:embed static_resources/skills
var embedded_skills embed.FS

const skillsRoot = "static_resources/skills"

func getEmbeddedSkills() []*Skill {
	skills := make([]*Skill, 0)

	entries, err := embedded_skills.ReadDir(skillsRoot)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		rootPath := path.Join(skillsRoot, e.Name())

		files := make([]string, 0)
		err := fs.WalkDir(embedded_skills, rootPath, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			files = append(files, p)
			return nil
		})
		if err != nil {
			fmt.Println(err)
			continue
		}

		s := &Skill{
			Name:        e.Name(),
			SkillFsPath: rootPath,
			Files:       files,
			fs:          embedded_skills,
		}

		skills = append(skills, s)
	}

	return skills
}
