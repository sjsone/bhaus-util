package skill

import (
	"io/fs"
)

type Skill struct {
	Name        string
	SkillFsPath string   // path of the skills root directory within fs
	Files       []string // paths of every file within the skill dir (SKILL.md, references/*, etc...)
	fs          fs.FS
}

func GetAvailableSkills() []*Skill {
	skills := make([]*Skill, 0)

	for _, s := range getEmbeddedSkills() {
		skills = append(skills, s)
	}

	return skills
}
