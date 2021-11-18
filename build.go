package nodemodulebom

import (
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	Generate(workingDir, layersDir, layerName string) error
}

func Build(sbom SBOMGenerator, clock chronos.Clock, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		sbomLayer, err := context.Layers.Get("node-module-sbom")
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Executing build process")
		sbomLayer, err = sbomLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}
		sbomLayer.Metadata = map[string]interface{}{
			"built_at": clock.Now().Format(time.RFC3339Nano),
		}

		sbomLayer.Launch = true

		duration, err := clock.Measure(func() error {
			//TODO: Pass path to lockfile instead of working dir root
			err = sbom.Generate(context.WorkingDir, context.Layers.Path, "node-module-sbom")
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		return packit.BuildResult{
			Layers: []packit.Layer{sbomLayer},
		}, nil
	}
}
