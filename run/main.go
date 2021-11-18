package main

import (
	"os"

	nodemodulebom "github.com/paketo-buildpacks/node-module-bom"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {

	packit.Run(
		nodemodulebom.Detect(),
		nodemodulebom.Build(
			nodemodulebom.NewModuleSBOM(pexec.NewExecutable("syft"), scribe.NewEmitter(os.Stdout)),
			chronos.DefaultClock,
			scribe.NewEmitter(os.Stdout),
		),
	)
}
