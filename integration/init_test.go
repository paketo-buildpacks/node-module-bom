package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ForestEckhardt/freezer"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	nodeModuleBOMBuildpack        string
	offlineNodeModuleBOMBuildpack string
	nodeEngineBuildpack           string
	offlineNodeEngineBuildpack    string
	npmInstallBuildpack           string
	yarnBuildpack                 string
	yarnInstallBuildpack          string
	nodeStartBuildpack            string
	npmStartBuildpack             string
	yarnStartBuildpack            string
	syftBuildpack                 string
	offlineSyftBuildpack          string
	root                          string

	config struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}

	integrationjson struct {
		NodeEngine string `json:"node-engine"`
		NodeStart  string `json:"node-start"`

		NPMInstall string `json:"npm-install"`
		NPMStart   string `json:"npm-start"`

		Yarn        string `json:"yarn"`
		YarnInstall string `json:"yarn-install"`
		YarnStart   string `json:"yarn-start"`
		Syft        string `json:"syft"`
	}
)

type createPackage struct {
	executable freezer.Executable
}

func NewCreatePackage() createPackage {
	return createPackage{
		executable: pexec.NewExecutable("create-package"),
	}
}

func (cp createPackage) Execute(buildpackDir, output, version string, cached bool) error {
	args := []string{
		"--destination", output,
		"--version", version,
	}

	if cached {
		args = append(args, "--include-dependencies")
	}

	return cp.executable.Execute(pexec.Execution{
		Args:   args,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Dir:    buildpackDir,
	})
}

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	var err error
	root, err = filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	file, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	_, err = toml.DecodeReader(file, &config)
	Expect(err).NotTo(HaveOccurred())

	file, err = os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())

	Expect(json.NewDecoder(file).Decode(&integrationjson)).To(Succeed())
	Expect(file.Close()).To(Succeed())

	buildpackStore := occam.NewBuildpackStore()

	cp := NewCreatePackage()
	createPackageBuildpackStore := occam.NewBuildpackStore().WithPackager(cp)

	nodeModuleBOMBuildpack, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	offlineNodeModuleBOMBuildpack, err = buildpackStore.Get.
		WithOfflineDependencies().
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	nodeEngineBuildpack, err = buildpackStore.Get.
		Execute(integrationjson.NodeEngine)
	Expect(err).NotTo(HaveOccurred())

	offlineNodeEngineBuildpack, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(integrationjson.NodeEngine)
	Expect(err).NotTo(HaveOccurred())

	nodeStartBuildpack, err = buildpackStore.Get.
		Execute(integrationjson.NodeStart)
	Expect(err).NotTo(HaveOccurred())

	npmInstallBuildpack, err = buildpackStore.Get.
		Execute(integrationjson.NPMInstall)
	Expect(err).NotTo(HaveOccurred())

	npmStartBuildpack, err = buildpackStore.Get.
		Execute(integrationjson.NPMStart)
	Expect(err).NotTo(HaveOccurred())

	yarnBuildpack, err = buildpackStore.Get.
		Execute(integrationjson.Yarn)
	Expect(err).NotTo(HaveOccurred())

	yarnInstallBuildpack, err = buildpackStore.Get.
		Execute(integrationjson.YarnInstall)
	Expect(err).NotTo(HaveOccurred())

	yarnStartBuildpack, err = buildpackStore.Get.
		Execute(integrationjson.YarnStart)
	Expect(err).NotTo(HaveOccurred())

	// syftBuildpack, err = buildpackStore.Get.
	// 	Execute(integrationjson.Syft)
	// Expect(err).NotTo(HaveOccurred())

	// offlineSyftBuildpack , err = createPackageBuildpackStore.Get.
	// WithOfflineDependencies().
	// 	Execute(integrationjson.Syft)
	// Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(5 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("NPM", testNPM)
	suite("Offline", testOffline)
	suite("PackageLockHashes", testPackageLockHashes)
	suite("Vendored", testVendored)
	suite("Yarn", testYarn)
	suite.Run(t)
}
