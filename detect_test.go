package nodemodulebom_test

import (
	"path/filepath"
	"testing"

	"os"

	nodemodulebom "github.com/paketo-buildpacks/node-module-bom"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		detect     packit.DetectFunc
		workingDir string
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		detect = nodemodulebom.Detect()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a plan that provides nothing and requires node and node_modules", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Plan).To(Equal(packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: "node",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
				{
					Name: "node_modules",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
			},
		}))
	})

	context("the app contains vendored node_modules", func() {
		it.Before(func() {
			Expect(os.Mkdir(filepath.Join(workingDir, "node_modules"), os.ModePerm)).To(Succeed())
		})

		it("returns a plan that provides nothing and only requires node", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "node",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		context("node_modules directory exists but cannot be stat", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})
	})

}
