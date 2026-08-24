// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/skaphos/repokeeper/test/e2e/internal/compatibility"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func compatibilityDeclarationPath() string {
	return filepath.Join(moduleRoot, "test", "e2e", "testdata", "git-compatibility.json")
}

var _ = Describe("Closed Git compatibility declaration", func() {
	It("strictly loads the complete ordered matrix and agrees with DESIGN.md", func() {
		declaration, err := compatibility.Load(compatibilityDeclarationPath())
		Expect(err).NotTo(HaveOccurred())
		Expect(declaration.SupportedMinors).To(Equal([]string{"2.53", "2.54", "2.55"}))
		Expect(declaration.Cells).To(HaveLen(12))
		routine, err := compatibility.Expand(declaration, "routine")
		Expect(err).NotTo(HaveOccurred())
		Expect(routine.Cells).To(HaveLen(4))
		release, err := compatibility.Expand(declaration, "release")
		Expect(err).NotTo(HaveOccurred())
		Expect(release.Cells).To(HaveLen(12))
		design, err := os.ReadFile(filepath.Join(moduleRoot, "DESIGN.md"))
		Expect(err).NotTo(HaveOccurred())
		Expect(compatibility.VerifyDocs(declaration, design)).To(Succeed())
	})

	DescribeTable("rejects compatibility contract drift", func(mutate func(*compatibility.Declaration), field string) {
		declaration, err := compatibility.Load(compatibilityDeclarationPath())
		Expect(err).NotTo(HaveOccurred())
		mutate(&declaration)
		Expect(compatibility.Validate(declaration)).To(MatchError(ContainSubstring(field)))
	},
		Entry("unordered minors", func(d *compatibility.Declaration) {
			d.SupportedMinors[0], d.SupportedMinors[1] = d.SupportedMinors[1], d.SupportedMinors[0]
		}, "supported_minors"),
		Entry("missing cell", func(d *compatibility.Declaration) { d.Cells = d.Cells[:11] }, "complete 4x3"),
		Entry("duplicate cell", func(d *compatibility.Declaration) { d.Cells[1] = d.Cells[0] }, "duplicate cell"),
		Entry("floating runner", func(d *compatibility.Declaration) { d.Cells[0].RunnerLabel = "ubuntu-latest" }, "runner_label"),
		Entry("patch mismatch", func(d *compatibility.Declaration) { d.Cells[0].GitPatch = "2.54.0" }, "git_patch"),
		Entry("provisioner mismatch", func(d *compatibility.Declaration) { d.Cells[0].Provisioner.Kind = "mingit-archive" }, "provisioner.kind"),
		Entry("missing WSL rootfs", func(d *compatibility.Declaration) {
			for i := range d.Cells {
				if d.Cells[i].Environment == "wsl" {
					d.Cells[i].RootFS = nil
					break
				}
			}
		}, "rootfs"),
		Entry("two routine Linux cells", func(d *compatibility.Declaration) { d.Cells[0].RoutineCI = true }, "routine_ci"),
	)

	It("rejects unknown and trailing JSON fields", func() {
		data, err := os.ReadFile(compatibilityDeclarationPath())
		Expect(err).NotTo(HaveOccurred())
		var object map[string]any
		Expect(json.Unmarshal(data, &object)).To(Succeed())
		object["unknown"] = true
		unknown, _ := json.Marshal(object)
		path := filepath.Join(GinkgoT().TempDir(), "unknown.json")
		Expect(os.WriteFile(path, unknown, 0o600)).To(Succeed())
		_, err = compatibility.Load(path)
		Expect(err).To(MatchError(ContainSubstring("unknown field")))
		Expect(os.WriteFile(path, append(data, []byte(" {}")...), 0o600)).To(Succeed())
		_, err = compatibility.Load(path)
		Expect(err).To(MatchError(ContainSubstring("trailing JSON")))
	})

	It("uses only immutable HTTPS inputs with lowercase SHA-256", func() {
		declaration, err := compatibility.Load(compatibilityDeclarationPath())
		Expect(err).NotTo(HaveOccurred())
		for _, cell := range declaration.Cells {
			Expect(cell.Provisioner.SourceURL).To(HavePrefix("https://"))
			Expect(cell.Provisioner.SHA256).To(HaveLen(64))
			Expect(cell.Provisioner.SHA256).To(Equal(strings.ToLower(cell.Provisioner.SHA256)))
		}
	})
})
