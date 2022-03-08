package main

import (
	"os"

	nodemodulebom "github.com/paketo-buildpacks/node-module-bom"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func main() {

	packit.Run(
		nodemodulebom.Detect(),
		nodemodulebom.Build(
			postal.NewService(cargo.NewTransport()),
			nodemodulebom.NewModuleBOM(pexec.NewExecutable("cyclonedx-bom"), scribe.NewEmitter(os.Stdout)),
			chronos.DefaultClock,
			scribe.NewEmitter(os.Stdout),
		),
	)
}
