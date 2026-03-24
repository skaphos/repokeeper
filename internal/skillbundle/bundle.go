// SPDX-License-Identifier: MIT
package skillbundle

import _ "embed"

const RepoKeeperSkillName = "repokeeper"

//go:embed repokeeper/SKILL.md
var repoKeeperSkill string

func RepoKeeperSkill() string {
	return repoKeeperSkill
}
