package nodemodulebom

import (
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/sbom"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

func Build(dependencyManager DependencyManager, clock chronos.Clock, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Process("Generating SBOM for directory %s", context.WorkingDir)

		var (
			bom sbom.SBOM
			err error
		)
		duration, err := clock.Measure(func() error {
			bom, err = sbom.Generate(context.WorkingDir)
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		layer, err := context.Layers.Get("node-module-bom")
		if err != nil {
			return packit.BuildResult{}, err
		}

		layer, err = layer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		layer.Launch = true

		layer.SBOM.Set("cdx.json", bom.Format(sbom.CycloneDXFormat))
		layer.SBOM.Set("syft.json", bom.Format(sbom.SyftFormat))
		layer.SBOM.Set("spdx.json", bom.Format(sbom.SPDXFormat))

		bomEntries := make(packit.SBOMEntries)
		bomEntries.Set("cdx.json", bom.Format(sbom.CycloneDXFormat))
		bomEntries.Set("syft.json", bom.Format(sbom.SyftFormat))
		bomEntries.Set("spdx.json", bom.Format(sbom.SPDXFormat))

		return packit.BuildResult{
			Layers: []packit.Layer{
				layer,
			},
			Build: packit.BuildMetadata{},
			Launch: packit.LaunchMetadata{
				SBOM: bomEntries,
			},
		}, nil
	}
}
