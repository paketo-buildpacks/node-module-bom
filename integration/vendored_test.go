package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testVendored(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when the buildpack is run with pack build", func() {
		var (
			image      occam.Image
			container1 occam.Container
			container2 occam.Container
			name       string
			source     string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		context("default vendored app builds", func() {
			it.After(func() {
				Expect(docker.Container.Remove.Execute(container1.ID)).To(Succeed())
				Expect(docker.Container.Remove.Execute(container2.ID)).To(Succeed())
			})

			it("builds, logs and runs correctly", func() {
				var err error

				source, err = occam.Source(filepath.Join("testdata", "vendored_app"))
				Expect(err).ToNot(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						nodeEngineBuildpack,
						nodeModuleBOMBuildpack,
						nodeStartBuildpack,
					).
					Execute(name, source)
				Expect(err).ToNot(HaveOccurred(), logs.String)

				Expect(logs).To(ContainLines(
					fmt.Sprintf("%s 1.2.3", config.Buildpack.Name),
					"  Resolving CycloneDX Node.js Module version",
					MatchRegexp(`    Selected CycloneDX Node.js Module version: \d+\.\d+\.\d+`),
					"",
					"  Executing build process",
					MatchRegexp(`    Installing CycloneDX Node.js Module \d+\.\d+\.\d+`),
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
					"",
					"  Configuring environment",
					"    Appending CycloneDX Node.js Module onto PATH",
					"",
					"  Running CycloneDX Node.js Module",
					`    Running 'cyclonedx-bom -o bom.json'`,
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				))

				container1, err = docker.Container.Run.
					WithPublish("8080").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container1).Should(BeAvailable())
				Eventually(container1).Should(Serve(ContainSubstring("hello world")).OnPort(8080))

				container2, err = docker.Container.Run.
					WithCommand("cat /layers/sbom/launch/sbom.legacy.json").
					WithEntrypoint("launcher").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() string {
					cLogs, err := docker.Container.Logs.Execute(container2.ID)
					Expect(err).NotTo(HaveOccurred())
					return cLogs.String()
				}).Should(ContainSubstring(`"name":"leftpad"`))
			})
		})

		context("when BP_DISABLE_SBOM is true", func() {
			it.After(func() {
				Expect(docker.Container.Remove.Execute(container1.ID)).To(Succeed())
				Expect(docker.Container.Remove.Execute(container2.ID)).To(Succeed())
			})

			it("skips SBOM generation", func() {
				var err error

				source, err = occam.Source(filepath.Join("testdata", "vendored_app"))
				Expect(err).ToNot(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithEnv(map[string]string{"BP_DISABLE_SBOM": "true"}).
					WithBuildpacks(
						nodeEngineBuildpack,
						nodeModuleBOMBuildpack,
						nodeStartBuildpack,
					).
					Execute(name, source)
				Expect(err).ToNot(HaveOccurred(), logs.String)

				Expect(logs).To(ContainSubstring("Skipping Node Module BOM generation"))

				Expect(logs).ToNot(ContainLines(
					"  Running CycloneDX Node.js Module",
					`    Running 'cyclonedx-bom -o bom.json'`,
				))

				container1, err = docker.Container.Run.
					WithPublish("8080").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container1).Should(BeAvailable())
				Eventually(container1).Should(Serve(ContainSubstring("hello world")).OnPort(8080))

				container2, err = docker.Container.Run.
					WithCommand("cat /layers/sbom/launch/sbom.legacy.json").
					WithEntrypoint("launcher").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() string {
					cLogs, err := docker.Container.Logs.Execute(container2.ID)
					Expect(err).NotTo(HaveOccurred())
					return cLogs.String()
				}).Should(Not(ContainSubstring(`"name":"leftpad"`)))
			})
		})
	})
}
