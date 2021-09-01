package nodemodulebom_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitNodeModuleBOM(t *testing.T) {
	suite := spec.New("node-module-bom", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("ModuleBOM", testModuleBOM)
	suite("PackageLock", testPackageLock)
	suite.Run(t)
}
