package nodemodulebom_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	nodemodulebom "github.com/paketo-buildpacks/node-module-bom"
	"github.com/paketo-buildpacks/node-module-bom/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testModuleBOM(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir    string
		executable    *fakes.Executable
		buffer        *bytes.Buffer
		commandOutput *bytes.Buffer

		moduleBOM nodemodulebom.ModuleBOM
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		executable = &fakes.Executable{}
		executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
			Expect(os.WriteFile(filepath.Join(workingDir, "bom.json"), []byte(
				`
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.3",
  "serialNumber": "urn:uuid:a717bde3-8a77-4ec6-a530-5d0d9007ecbe",
  "version": 1,
  "metadata": {
    "timestamp": "2021-08-16T19:35:52.107Z",
    "tools": [
      {
        "vendor": "CycloneDX",
        "name": "Node.js module",
        "version": "3.0.3"
      }
    ],
    "component": {
      "type": "library"
    }
  },
  "components": [
    {
      "type": "library",
      "name": "leftpad",
      "version": "0.0.1",
      "description": "left pad numbers",
      "hashes": [
        {
          "alg": "SHA-1",
          "content": "86b1a4de4face180ac545a83f1503523d8fed115"
        }
      ],
      "licenses": [
        {
          "license": {
            "id": "BSD-3-Clause"
          }
        }
      ],
      "purl": "pkg:npm/leftpad@0.0.1"
		},
    {
      "type": "library",
      "name": "rightpad",
      "version": "1.0.0",
      "description": "right pad numbers",
      "licenses": [
        {
          "license": {
            "id": "Apache"
          }
        }
      ],
			"purl": "pkg:npm/rightpad@1.0.0"
    }
  ]
}
`), 0644)).To(Succeed())
			return nil
		}

		buffer = bytes.NewBuffer(nil)
		commandOutput = bytes.NewBuffer(nil)

		moduleBOM = nodemodulebom.NewModuleBOM(executable, scribe.NewEmitter(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Generate", func() {
		it("succeeds in installing the BOM generation tool", func() {
			bomEntries, err := moduleBOM.Generate(workingDir)
			Expect(err).ToNot(HaveOccurred())

			Expect(executable.ExecuteCall.Receives.Execution).To(Equal(pexec.Execution{
				Args:   []string{"-o", "bom.json"},
				Dir:    workingDir,
				Stdout: commandOutput,
				Stderr: commandOutput,
			}))

			algorithm, err := packit.GetBOMChecksumAlgorithm("SHA-1")
			Expect(err).ToNot(HaveOccurred())

			Expect(bomEntries).To(Equal([]packit.BOMEntry{
				{
					Name: "leftpad",
					Metadata: &packit.BOMMetadata{
						Checksum: &packit.BOMChecksum{
							Algorithm: algorithm,
							Hash:      "86b1a4de4face180ac545a83f1503523d8fed115",
						},
						PURL:     "pkg:npm/leftpad@0.0.1",
						Licenses: []string{"BSD-3-Clause"},
						Version:  "0.0.1",
					},
				},
				{
					Name: "rightpad",
					Metadata: &packit.BOMMetadata{
						PURL:     "pkg:npm/rightpad@1.0.0",
						Licenses: []string{"Apache"},
						Version:  "1.0.0",
					},
				},
			}))

			Expect(filepath.Join(workingDir, "bom.json")).ToNot(BeAnExistingFile())
		})

		context("failure cases", func() {
			context("the cyclonedx-bom executable call fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						fmt.Fprintln(execution.Stdout, "build error stdout")
						fmt.Fprintln(execution.Stderr, "build error stderr")
						return errors.New("error")
					}
				})
				it("returns an error", func() {
					_, err := moduleBOM.Generate(workingDir)
					Expect(err).To(MatchError("failed to run cyclonedx-bom: error"))

					Expect(buffer.String()).To(ContainSubstring("        build error stdout"))
					Expect(buffer.String()).To(ContainSubstring("        build error stderr"))
				})
			})

			context("cannot open the bom.json file", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						Expect(os.WriteFile(filepath.Join(workingDir, "bom.json"), []byte(``), 0000)).To(Succeed())
						return nil
					}
				})

				it("returns an error", func() {
					_, err := moduleBOM.Generate(workingDir)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("failed to open bom.json")))
				})
			})

			context("cannot decode the bom.json into a struct", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						Expect(os.WriteFile(filepath.Join(workingDir, "bom.json"), []byte(``), 0644)).To(Succeed())
						return nil
					}
				})

				it("returns an error", func() {
					_, err := moduleBOM.Generate(workingDir)
					Expect(err).To(MatchError(ContainSubstring("failed to decode bom.json")))
				})
			})

			context("the BOM entry contains unsupported checksum algorithm", func() {
				it.Before(func() {

					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						Expect(os.WriteFile(filepath.Join(workingDir, "bom.json"), []byte(
							`
{
  "components": [
    {
      "type": "library",
      "name": "leftpad",
      "version": "0.0.1",
      "description": "left pad numbers",
      "hashes": [
        {
          "alg": "randomAlgorithm",
          "content": "86b1a4de4face180ac545a83f1503523d8fed115"
        }
      ]
		}
  ]
}
`), 0644)).To(Succeed())
						return nil
					}

					buffer = bytes.NewBuffer(nil)
					commandOutput = bytes.NewBuffer(nil)

					moduleBOM = nodemodulebom.NewModuleBOM(executable, scribe.NewEmitter(buffer))
				})

				it.After(func() {
					Expect(os.RemoveAll(workingDir)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := moduleBOM.Generate(workingDir)
					Expect(err).To(MatchError("failed to get supported BOM checksum algorithm: randomAlgorithm is not valid"))

				})
			})
		})
	})
}
