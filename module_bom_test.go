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
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testModuleSBOM(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir    string
		layersDir     string
		executable    *fakes.Executable
		buffer        *bytes.Buffer
		commandOutput *bytes.Buffer
		moduleSBOM    nodemodulebom.ModuleSBOM
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		layersDir, err = ioutil.TempDir("", "layers-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		commandOutput = bytes.NewBuffer(nil)

		executable = &fakes.Executable{}
		moduleSBOM = nodemodulebom.NewModuleSBOM(executable, scribe.NewEmitter(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("Generate", func() {
		it("generates a syft sbom file in the provided layer directory", func() {
			err := moduleSBOM.Generate(workingDir, layersDir, "some-name")
			Expect(err).ToNot(HaveOccurred())

			Expect(executable.ExecuteCall.Receives.Execution).To(Equal(pexec.Execution{
				Args:   []string{"packages", fmt.Sprintf("dir:%s", workingDir), "--output", "json", "--file", filepath.Join(layersDir, "some-name.sbom.syft.json")},
				Dir:    workingDir,
				Stdout: commandOutput,
				Stderr: commandOutput,
			}))
		})

		context("failure cases", func() {
			context("when syft execution fails", func() {
				it.Before(func() {
					executable.ExecuteCall.Stub = func(execution pexec.Execution) error {
						fmt.Fprint(execution.Stderr, "output from syft")
						return errors.New("syft failure")
					}
				})

				it("returns the error", func() {
					err := moduleSBOM.Generate(workingDir, layersDir, "some-name")
					Expect(err).To(MatchError("syft failure"))
					Expect(buffer.String()).To(ContainSubstring("output from syft"))
				})
			})
		})
	})
}
