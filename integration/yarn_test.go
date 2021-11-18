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

func testYarn(t *testing.T, context spec.G, it spec.S) {
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

	context.Focus("when the buildpack is run with pack build", func() {
		var (
			image               occam.Image
			container           occam.Container
			layerSBOMContainer  occam.Container
			launchSBOMContainer occam.Container
			name                string
			source              string
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

		context("building a basic yarn app is pack built", func() {
			it.After(func() {
				Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
				Expect(docker.Container.Remove.Execute(layerSBOMContainer.ID)).To(Succeed())
				Expect(docker.Container.Remove.Execute(launchSBOMContainer.ID)).To(Succeed())
			})

			it("builds, logs and runs correctly", func() {
				var err error

				source, err = occam.Source(filepath.Join("testdata", "yarn_app"))
				Expect(err).ToNot(HaveOccurred())

				home, err := os.UserHomeDir()
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().WithVerbose().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						nodeEngineBuildpack,
						yarnBuildpack,
						yarnInstallBuildpack,
						// syftBuildpack,
						filepath.Join(home, "Downloads", "syft.tgz"),
						nodeModuleBOMBuildpack,
						yarnStartBuildpack,
					).
					Execute(name, source)
				Expect(err).ToNot(HaveOccurred(), logs.String)

				container, err = docker.Container.Run.
					WithPublish("8080").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())
				Eventually(container).Should(Serve(ContainSubstring("hello world")).OnPort(8080))

				layerSBOMContainer, err = docker.Container.Run.
					WithPublish("8080").
					WithEntrypoint("launcher").
					// WithCommand("cat /layers/sbom/paketo-buildpacks_node-module-bom/launch/node-module-bom/sbom.syft.json").
					// TODO: File an issue against lifecycle pointing out that the path above (derived from inline comment) is wrong
					// https://github.com/buildpacks/lifecycle/blob/main/builder.go#L143-L150
					WithCommand("cat  /layers/sbom/launch/paketo-buildpacks_node-module-bom/node-module-sbom/sbom.syft.json").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() string {
					cLogs, err := docker.Container.Logs.Execute(layerSBOMContainer.ID)
					Expect(err).NotTo(HaveOccurred())
					return cLogs.String()
				}).Should(ContainSubstring("leftpad"))

				launchSBOMContainer, err = docker.Container.Run.
					WithPublish("8080").
					WithEntrypoint("launcher").
					WithCommand("cat  /layers/sbom/launch/sbom.syft.json").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() string {
					cLogs, err := docker.Container.Logs.Execute(launchSBOMContainer.ID)
					Expect(err).NotTo(HaveOccurred())
					return cLogs.String()
				}).Should(ContainSubstring("leftpad"))
			})
		})
	})
}
