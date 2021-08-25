package nodemodulebom_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	nodemodulebom "github.com/paketo-buildpacks/node-module-bom"
	"github.com/paketo-buildpacks/node-module-bom/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir         string
		cnbDir            string
		workingDir        string
		timestamp         time.Time
		dependencyManager *fakes.DependencyManager
		nodeModuleBOM     *fakes.NodeModuleBOM
		buffer            *bytes.Buffer

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

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "cyclonedx-node-module",
			Name:    "cyclonedx-node-module-dependency-name",
			SHA256:  "cyclonedx-node-module-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "cyclonedx-node-module-dependency-uri",
			Version: "cyclonedx-node-module-dependency-version",
		}

		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "cyclonedx-node-module",
				Metadata: &packit.BOMMetadata{
					Version: "cyclonedx-node-module-dependency-version",
					Checksum: &packit.BOMChecksum{
						Algorithm: "SHA-256",
						Hash:      "cyclonedx-node-module-dependency-sha",
					},
					URI: "cyclonedx-node-module-dependency-uri",
				},
			},
		}

		nodeModuleBOM = &fakes.NodeModuleBOM{}
		nodeModuleBOM.GenerateCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "leftpad",
				Metadata: &packit.BOMMetadata{
					Version: "leftpad-dependency-version",
					Checksum: &packit.BOMChecksum{
						Algorithm: "SHA-256",
						Hash:      "leftpad-dependency-sha",
					},
					URI: "leftpad-dependency-uri",
				},
			},
		}

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		build = nodemodulebom.Build(dependencyManager, nodeModuleBOM, clock, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs cyclonedx-node-module", func() {
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
					Name:             "cyclonedx-node-module",
					Path:             filepath.Join(layersDir, "cyclonedx-node-module"),
					SharedEnv:        packit.Environment{},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           false,
					Cache:            true,
					Metadata: map[string]interface{}{
						"dependency-sha": "cyclonedx-node-module-dependency-sha",
						"built_at":       timestamp.Format(time.RFC3339Nano),
					},
				},
			},
			Build: packit.BuildMetadata{
				BOM: []packit.BOMEntry{
					{
						Name: "cyclonedx-node-module",
						Metadata: &packit.BOMMetadata{
							Version: "cyclonedx-node-module-dependency-version",
							Checksum: &packit.BOMChecksum{
								Algorithm: "SHA-256",
								Hash:      "cyclonedx-node-module-dependency-sha",
							},
							URI: "cyclonedx-node-module-dependency-uri",
						},
					},
					{
						Name: "leftpad",
						Metadata: &packit.BOMMetadata{
							Version: "leftpad-dependency-version",
							Checksum: &packit.BOMChecksum{
								Algorithm: "SHA-256",
								Hash:      "leftpad-dependency-sha",
							},
							URI: "leftpad-dependency-uri",
						},
					},
				},
			},
			Launch: packit.LaunchMetadata{
				BOM: []packit.BOMEntry{
					{
						Name: "leftpad",
						Metadata: &packit.BOMMetadata{
							Version: "leftpad-dependency-version",
							Checksum: &packit.BOMChecksum{
								Algorithm: "SHA-256",
								Hash:      "leftpad-dependency-sha",
							},
							URI: "leftpad-dependency-uri",
						},
					},
				},
			},
		}))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("cyclonedx-node-module"))
		Expect(dependencyManager.ResolveCall.Receives.Version).To(Equal("*"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "cyclonedx-node-module",
			Name:    "cyclonedx-node-module-dependency-name",
			SHA256:  "cyclonedx-node-module-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "cyclonedx-node-module-dependency-uri",
			Version: "cyclonedx-node-module-dependency-version",
		}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "cyclonedx-node-module")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:      "cyclonedx-node-module",
				Name:    "cyclonedx-node-module-dependency-name",
				SHA256:  "cyclonedx-node-module-dependency-sha",
				Stacks:  []string{"some-stack"},
				URI:     "cyclonedx-node-module-dependency-uri",
				Version: "cyclonedx-node-module-dependency-version",
			},
		}))

		Expect(nodeModuleBOM.GenerateCall.Receives.WorkingDir).To(Equal(workingDir))
	})

	context("when there is a dependency cache match to reuse", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(layersDir, "cyclonedx-node-module.toml"), []byte(`
			[metadata]
			dependency-sha = "cyclonedx-node-module-dependency-sha"
			`), 0644)
			Expect(err).NotTo(HaveOccurred())

			dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
				ID:      "cyclonedx-node-module",
				Name:    "cyclonedx-node-module-dependency-name",
				SHA256:  "cyclonedx-node-module-dependency-sha",
				Stacks:  []string{"some-stack"},
				URI:     "cyclonedx-node-module-dependency-uri",
				Version: "cyclonedx-node-module-dependency-version",
			}
		})

		it("reuses the cache", func() {
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
						Name:             "cyclonedx-node-module",
						Path:             filepath.Join(layersDir, "cyclonedx-node-module"),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            false,
						Launch:           false,
						Cache:            true,
						Metadata: map[string]interface{}{
							"dependency-sha": "cyclonedx-node-module-dependency-sha",
						},
					},
				},
				Build: packit.BuildMetadata{
					BOM: []packit.BOMEntry{
						{
							Name: "cyclonedx-node-module",
							Metadata: &packit.BOMMetadata{
								Version: "cyclonedx-node-module-dependency-version",
								Checksum: &packit.BOMChecksum{
									Algorithm: "SHA-256",
									Hash:      "cyclonedx-node-module-dependency-sha",
								},
								URI: "cyclonedx-node-module-dependency-uri",
							},
						},
						{
							Name: "leftpad",
							Metadata: &packit.BOMMetadata{
								Version: "leftpad-dependency-version",
								Checksum: &packit.BOMChecksum{
									Algorithm: "SHA-256",
									Hash:      "leftpad-dependency-sha",
								},
								URI: "leftpad-dependency-uri",
							},
						},
					},
				},
				Launch: packit.LaunchMetadata{
					BOM: []packit.BOMEntry{
						{
							Name: "leftpad",
							Metadata: &packit.BOMMetadata{
								Version: "leftpad-dependency-version",
								Checksum: &packit.BOMChecksum{
									Algorithm: "SHA-256",
									Hash:      "leftpad-dependency-sha",
								},
								URI: "leftpad-dependency-uri",
							},
						},
					},
				},
			}))

			Expect(dependencyManager.DeliverCall.CallCount).To(Equal(0))
			Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
				{
					ID:      "cyclonedx-node-module",
					Name:    "cyclonedx-node-module-dependency-name",
					SHA256:  "cyclonedx-node-module-dependency-sha",
					Stacks:  []string{"some-stack"},
					URI:     "cyclonedx-node-module-dependency-uri",
					Version: "cyclonedx-node-module-dependency-version",
				},
			}))
			Expect(nodeModuleBOM.GenerateCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
			Expect(buffer.String()).ToNot(ContainSubstring("Executing build process"))
		})
	})

	context("failure cases", func() {
		context("the dependency cannot be resolved", func() {
			it.Before(func() {
				dependencyManager.ResolveCall.Returns.Error = errors.New("failed to resolve dependency")
			})
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbDir,
					Platform:   packit.Platform{Path: "platform"},
					Layers:     packit.Layers{Path: layersDir},
					Stack:      "some-stack",
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to resolve dependency")))
			})
		})

		context("when the cyclonedx-node-module layer cannot be retrieved", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "cyclonedx-node-module.toml"), nil, 0000)
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

		context("when the cyclonedx-node-module layer cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "cyclonedx-node-module", "something"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(filepath.Join(layersDir, "cyclonedx-node-module"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "cyclonedx-node-module"), os.ModePerm)).To(Succeed())
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

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Returns.Error = errors.New("failed to install dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbDir,
					Platform:   packit.Platform{Path: "platform"},
					Layers:     packit.Layers{Path: layersDir},
					Stack:      "some-stack",
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to install dependency")))
			})
		})

		context("when the node module BOM cannot be generated", func() {
			it.Before(func() {
				nodeModuleBOM.GenerateCall.Returns.Error = errors.New("failed to generate node module BOM")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbDir,
					Platform:   packit.Platform{Path: "platform"},
					Layers:     packit.Layers{Path: layersDir},
					Stack:      "some-stack",
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to generate node module BOM")))
			})
		})
	})
}
