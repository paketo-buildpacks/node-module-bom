package nodemodulebom_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	nodemodulebom "github.com/paketo-buildpacks/node-module-bom"
	"github.com/paketo-buildpacks/node-module-bom/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		cnbDir     string
		workingDir string
		timestamp  time.Time
		buffer     *bytes.Buffer
		sbom       *fakes.SBOMGenerator

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "workingDir")
		Expect(err).NotTo(HaveOccurred())

		timestamp = time.Now()
		clock := chronos.NewClock(func() time.Time {
			return timestamp
		})

		sbom = &fakes.SBOMGenerator{}
		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		build = nodemodulebom.Build(sbom, clock, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that has an sbom layer", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			CNBPath:    cnbDir,
			Platform:   packit.Platform{Path: "platform"},
			Layers:     packit.Layers{Path: layersDir},
			Stack:      "some-stack",
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				{
					Name:             "node-module-sbom",
					Path:             filepath.Join(layersDir, "node-module-sbom"),
					SharedEnv:        packit.Environment{},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           true,
					Cache:            false,
					Metadata: map[string]interface{}{
						"built_at": timestamp.Format(time.RFC3339Nano),
					},
				},
			},
			Build:  packit.BuildMetadata{},
			Launch: packit.LaunchMetadata{},
		}))

		Expect(sbom.GenerateCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(sbom.GenerateCall.Receives.LayersDir).To(Equal(layersDir))
		Expect(sbom.GenerateCall.Receives.LayerName).To(Equal("node-module-sbom"))
	})

	context("failure cases", func() {
		context("when the node-module-sbom layer cannot be retrieved", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "node-module-sbom.toml"), nil, 0000)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbDir,
					Platform:   packit.Platform{Path: "platform"},
					Layers:     packit.Layers{Path: layersDir},
					Stack:      "some-stack",
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
			})
		})

		context("when the node-module-sbom layer cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "node-module-sbom", "something"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(filepath.Join(layersDir, "node-module-sbom"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "node-module-sbom"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbDir,
					Platform:   packit.Platform{Path: "platform"},
					Layers:     packit.Layers{Path: layersDir},
					Stack:      "some-stack",
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})
		context("when sbom generation fails", func() {
			it.Before(func() {
				sbom.GenerateCall.Returns.Error = errors.New("some sbom error")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbDir,
					Platform:   packit.Platform{Path: "platform"},
					Layers:     packit.Layers{Path: layersDir},
					Stack:      "some-stack",
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("some sbom error")))
			})
		})
	})
}
