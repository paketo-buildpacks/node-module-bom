package nodemodulebom_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNodeModuleSBOM(t *testing.T) {
	suite := spec.New("node-module-sbom", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("ModuleSBOM", testModuleSBOM)
	suite.Run(t)
}
