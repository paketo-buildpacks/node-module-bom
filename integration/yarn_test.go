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

		context("building a basic yarn app is pack built", func() {
			it.After(func() {
				Expect(docker.Container.Remove.Execute(container1.ID)).To(Succeed())
				Expect(docker.Container.Remove.Execute(container2.ID)).To(Succeed())
			})

			it("builds, logs and runs correctly", func() {
				var err error

				source, err = occam.Source(filepath.Join("testdata", "yarn_app"))
				Expect(err).ToNot(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						nodeEngineBuildpack,
						yarnBuildpack,
						yarnInstallBuildpack,
						nodeModuleBOMBuildpack,
						yarnStartBuildpack,
					).
					Execute(name, source)
				Expect(err).ToNot(HaveOccurred(), logs.String)

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
	})
}
