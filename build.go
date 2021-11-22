package nodemodulebom

import (
	"io/ioutil"
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

//go:generate faux --interface NodeModuleBOM --output fakes/node_module_bom.go
type NodeModuleBOM interface {
	Generate(workingDir string) ([]packit.BOMEntry, error)
}

func Build(dependencyManager DependencyManager, nodeModuleBOM NodeModuleBOM, clock chronos.Clock, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Process("Generating SBOM for directory %s", context.WorkingDir)

		files, err := ioutil.ReadDir(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Detail("contents of: %s", context.WorkingDir)
		for _, f := range files {
			logger.Detail(f.Name())
		}

		var bom sbom.SBOM
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
		layer.SBOM.Set("cdx.json", bom.Format(sbom.CycloneDXFormat))
		layer.SBOM.Set("syft.json", bom.Format(sbom.SyftFormat))
		layer.SBOM.Set("spdx.json", bom.Format(sbom.SPDXFormat))

		b, err := ioutil.ReadAll(bom.Format(sbom.SyftFormat))
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Detail("%s", string(b[:]))

		return packit.BuildResult{
			Layers: []packit.Layer{
				layer,
			},
			Build: packit.BuildMetadata{
				SBOM: layer.SBOM,
			},
			Launch: packit.LaunchMetadata{
				SBOM: layer.SBOM,
			},
		}, nil
	}
}
