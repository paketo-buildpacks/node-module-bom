package nodemodulebom_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	nodemodulebom "github.com/paketo-buildpacks/node-module-bom"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPackageLock(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		lockfile   nodemodulebom.PackageLock
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte(`
			{
				"name": "simple_app",
				"lockfileVersion": 1,
				"requires": true,
				"dependencies": {
					"leftpad": {
						"version": "0.0.1",
						"integrity": "sha256-YWJjZGU="
					}
				}
			}`), 0600)).To(Succeed())

		lockfile, err = nodemodulebom.NewPackageLock(workingDir)
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("FindChecksum", func() {
		it("successfully retrieves checksum and returns a packit.BOMChecksum", func() {
			checksum, err := lockfile.FindChecksum("leftpad")
			Expect(err).ToNot(HaveOccurred())

			Expect(checksum).To(Equal(packit.BOMChecksum{
				Algorithm: packit.SHA256,
				Hash:      "6162636465",
			}))

			context("package-lock.json does not contain matching dependency", func() {
				it("returns an empty BOMChecksum", func() {
					checksum, err := lockfile.FindChecksum("another-dependency")
					Expect(err).ToNot(HaveOccurred())
					Expect(checksum).To(Equal(packit.BOMChecksum{}))
				})
			})

			context("package-lock.json integrity is not formatted with '-'", func() {
				it.Before(func() {
					var err error
					Expect(os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte(`
						{
							"name": "simple_app",
							"lockfileVersion": 1,
							"requires": true,
							"dependencies": {
								"leftpad": {
									"version": "0.0.1",
									"integrity": "algorithmHash"
								}
							}
						}`), 0600)).To(Succeed())
					lockfile, err = nodemodulebom.NewPackageLock(workingDir)
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(workingDir)).To(Succeed())
				})

				it("returns an empty BOMChecksum", func() {
					checksum, err := lockfile.FindChecksum("leftpad")
					Expect(err).ToNot(HaveOccurred())
					Expect(checksum).To(Equal(packit.BOMChecksum{}))
				})
			})
		})

		context("failure cases", func() {
			context("cannot decode hash string", func() {
				it.Before(func() {
					var err error
					Expect(os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte(`
						{
							"name": "simple_app",
							"lockfileVersion": 1,
							"requires": true,
							"dependencies": {
								"leftpad": {
									"version": "0.0.1",
									"integrity": "sha256-%%"
								}
							}
						}`), 0600)).To(Succeed())
					lockfile, err = nodemodulebom.NewPackageLock(workingDir)
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(workingDir)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := lockfile.FindChecksum("leftpad")
					Expect(err).To(MatchError(ContainSubstring("illegal base64 data at input byte 0")))
				})
			})

			context("cannot get a packit.BOMChecksumAlgorithm from algorithm", func() {
				it.Before(func() {
					var err error
					Expect(os.WriteFile(filepath.Join(workingDir, "package-lock.json"), []byte(`
						{
							"name": "simple_app",
							"lockfileVersion": 1,
							"requires": true,
							"dependencies": {
								"leftpad": {
									"version": "0.0.1",
									"integrity": "randomAlg-YWJjZGU="
								}
							}
						}`), 0600)).To(Succeed())
					lockfile, err = nodemodulebom.NewPackageLock(workingDir)
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(workingDir)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := lockfile.FindChecksum("leftpad")
					Expect(err).To(MatchError(ContainSubstring("failed to get supported BOM checksum algorithm: randomAlg is not valid")))
				})
			})
		})

	})
}
