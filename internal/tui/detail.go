// SPDX-License-Identifier: MIT
package tui

import (
	"fmt"
	"strings"

	"github.com/skaphos/repokeeper/internal/model"
)

func renderDetailView(m tuiModel) string {
	list := m.visibleList()
	if len(list) == 0 || m.cursor >= len(list) {
		return renderListView(m)
	}
	r := list[m.cursor]

	var b strings.Builder
	b.WriteString(titleStyle.Render("Repository: " + r.RepoID))
	b.WriteByte('\n')
	b.WriteString(renderDivider([]int{m.width - 1}))
	b.WriteByte('\n')

	field := func(label, value string) {
		fmt.Fprintf(&b, "  %-18s %s\n", label+":", value)
	}

	field("Path", r.Path)
	field("Type", repoType(r))
	field("Branch", colValueBranch(r))
	field("Status", colValueStatus(r))
	field("Delta", deltaOrDash(r))
	field("Dirty", dirtyDisplay(r))
	field("Error", errorDisplay(r))
	field("Upstream", upstreamDisplay(r))
	b.WriteByte('\n')

	if len(r.Remotes) > 0 {
		b.WriteString(headerStyle.Render("Remotes"))
		b.WriteByte('\n')
		for _, rem := range r.Remotes {
			fmt.Fprintf(&b, "  %-10s %s\n", rem.Name, rem.URL)
		}
		b.WriteByte('\n')
	}

	if len(r.Labels) > 0 {
		b.WriteString(headerStyle.Render("Labels"))
		b.WriteByte('\n')
		for k, v := range r.Labels {
			fmt.Fprintf(&b, "  %s=%s\n", k, v)
		}
		b.WriteByte('\n')
	}

	if len(r.Annotations) > 0 {
		b.WriteString(headerStyle.Render("Annotations"))
		b.WriteByte('\n')
		for k, v := range r.Annotations {
			fmt.Fprintf(&b, "  %s=%s\n", k, v)
		}
		b.WriteByte('\n')
	}

	if r.RepoMetadataFile != "" || r.RepoMetadata != nil || r.RepoMetadataError != "" {
		b.WriteString(headerStyle.Render("Repo Metadata"))
		b.WriteByte('\n')
		if r.RepoMetadataFile != "" {
			fmt.Fprintf(&b, "  File: %s\n", r.RepoMetadataFile)
		}
		if r.RepoMetadata != nil {
			if r.RepoMetadata.Name != "" {
				fmt.Fprintf(&b, "  Name: %s\n", r.RepoMetadata.Name)
			}
			if r.RepoMetadata.RepoID != "" {
				fmt.Fprintf(&b, "  Repo ID: %s\n", r.RepoMetadata.RepoID)
			}
			if len(r.RepoMetadata.Labels) > 0 {
				b.WriteString("  Labels:\n")
				for k, v := range r.RepoMetadata.Labels {
					fmt.Fprintf(&b, "    %s=%s\n", k, v)
				}
			}
			if len(r.RepoMetadata.Entrypoints) > 0 {
				b.WriteString("  Entrypoints:\n")
				for k, v := range r.RepoMetadata.Entrypoints {
					fmt.Fprintf(&b, "    %s=%s\n", k, v)
				}
			}
			if len(r.RepoMetadata.Paths.Authoritative) > 0 {
				fmt.Fprintf(&b, "  Authoritative: %s\n", strings.Join(r.RepoMetadata.Paths.Authoritative, ", "))
			}
			if len(r.RepoMetadata.Paths.LowValue) > 0 {
				fmt.Fprintf(&b, "  Low value: %s\n", strings.Join(r.RepoMetadata.Paths.LowValue, ", "))
			}
			if len(r.RepoMetadata.Provides) > 0 {
				fmt.Fprintf(&b, "  Provides: %s\n", strings.Join(r.RepoMetadata.Provides, ", "))
			}
			if len(r.RepoMetadata.RelatedRepos) > 0 {
				b.WriteString("  Related repos:\n")
				for _, related := range r.RepoMetadata.RelatedRepos {
					if related.Relationship == "" {
						fmt.Fprintf(&b, "    %s\n", related.RepoID)
						continue
					}
					fmt.Fprintf(&b, "    %s (%s)\n", related.RepoID, related.Relationship)
				}
			}
		}
		if r.RepoMetadataError != "" {
			fmt.Fprintf(&b, "  Error: %s\n", r.RepoMetadataError)
		}
		b.WriteByte('\n')
	}

	if r.LastSync != nil {
		b.WriteString(headerStyle.Render("Last Sync"))
		b.WriteByte('\n')
		ok := "✓"
		if !r.LastSync.OK {
			ok = "✗"
		}
		fmt.Fprintf(&b, "  %s  %s\n", ok, relativeTime(r.LastSync.At))
		if r.LastSync.Error != "" {
			fmt.Fprintf(&b, "  Error: %s\n", r.LastSync.Error)
		}
		b.WriteByte('\n')
	}

	b.WriteString(statusBarStyle.Render("esc/q: back  e: edit metadata  r: repair upstream"))
	return b.String()
}

func repoType(r model.RepoStatus) string {
	if r.Type == "" {
		return "checkout"
	}
	return r.Type
}

func deltaOrDash(r model.RepoStatus) string {
	v := colValueDelta(r)
	if v == "" {
		return "-"
	}
	return v
}

func dirtyDisplay(r model.RepoStatus) string {
	if r.Bare {
		return "bare (no worktree)"
	}
	if r.Worktree == nil {
		return "-"
	}
	if r.Worktree.Dirty {
		return fmt.Sprintf("yes (staged:%d unstaged:%d untracked:%d)",
			r.Worktree.Staged, r.Worktree.Unstaged, r.Worktree.Untracked)
	}
	return "clean"
}

func errorDisplay(r model.RepoStatus) string {
	if r.Error == "" {
		return "-"
	}
	if r.ErrorClass != "" {
		return fmt.Sprintf("[%s] %s", r.ErrorClass, r.Error)
	}
	return r.Error
}

func upstreamDisplay(r model.RepoStatus) string {
	if r.Tracking.Upstream == "" {
		return "-"
	}
	return r.Tracking.Upstream
}
