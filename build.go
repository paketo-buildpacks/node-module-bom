package nodemodulebom

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
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

		logger.Process("Resolving CycloneDX Node.js Module version")

		dependency, err := dependencyManager.Resolve(
			filepath.Join(context.CNBPath, "buildpack.toml"),
			"cyclonedx-node-module",
			"*",
			context.Stack,
		)
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Subprocess("Selected %s version: %s", dependency.Name, dependency.Version)
		logger.Break()

		cycloneDXNodeModuleLayer, err := context.Layers.Get("cyclonedx-node-module")
		if err != nil {
			return packit.BuildResult{}, err
		}

		cachedSHA, ok := cycloneDXNodeModuleLayer.Metadata["dependency-sha"].(string)
		if ok && cachedSHA == dependency.SHA256 { //nolint:staticcheck
			logger.Process("Reusing cached layer %s", cycloneDXNodeModuleLayer.Path)
			logger.Break()
		} else {
			logger.Process("Executing build process")
			cycloneDXNodeModuleLayer, err = cycloneDXNodeModuleLayer.Reset()
			if err != nil {
				return packit.BuildResult{}, err
			}
			logger.Subprocess("Installing %s %s", dependency.Name, dependency.Version)
			duration, err := clock.Measure(func() error {
				return dependencyManager.Deliver(dependency, context.CNBPath, cycloneDXNodeModuleLayer.Path, context.Platform.Path)
			})
			if err != nil {
				return packit.BuildResult{}, err
			}

			logger.Action("Completed in %s", duration.Round(time.Millisecond))
			logger.Break()

			cycloneDXNodeModuleLayer.Metadata = map[string]interface{}{
				"dependency-sha": dependency.SHA256, //nolint:staticcheck
			}
		}

		cycloneDXNodeModuleLayer.Cache = true

		logger.Process("Configuring environment")
		logger.Subprocess("Appending %s onto PATH", dependency.Name)
		logger.Break()

		os.Setenv("PATH", fmt.Sprint(os.Getenv("PATH"), string(os.PathListSeparator), filepath.Join(cycloneDXNodeModuleLayer.Path, "bin")))

		sbomDisabled, err := checkSbomDisabled()
		if err != nil {
			return packit.BuildResult{}, err
		}

		var toolBOM, moduleBOM []packit.BOMEntry

		if sbomDisabled {
			logger.Subprocess("Skipping Node Module BOM generation")
			logger.Break()
		} else {
			toolBOM = dependencyManager.GenerateBillOfMaterials(dependency)

			logger.Process("Running %s", dependency.Name)

			duration, err := clock.Measure(func() error {
				moduleBOM, err = nodeModuleBOM.Generate(context.WorkingDir)
				return err
			})
			if err != nil {
				return packit.BuildResult{}, err
			}

			logger.Action("Completed in %s", duration.Round(time.Millisecond))
			logger.Break()
		}

		return packit.BuildResult{
			Layers: []packit.Layer{cycloneDXNodeModuleLayer},
			Build: packit.BuildMetadata{
				BOM: append(toolBOM, moduleBOM...),
			},
			Launch: packit.LaunchMetadata{
				BOM: moduleBOM,
			},
		}, nil
	}
}

func checkSbomDisabled() (bool, error) {
	if disableStr, ok := os.LookupEnv("BP_DISABLE_SBOM"); ok {
		disable, err := strconv.ParseBool(disableStr)
		if err != nil {
			return false, fmt.Errorf("failed to parse BP_DISABLE_SBOM value %s: %w", disableStr, err)
		}
		return disable, nil
	}
	return false, nil
}
