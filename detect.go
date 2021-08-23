package nodemodulebom

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.BuildPlan{
			Requires: []packit.BuildPlanRequirement{
				{
					Name: "node",
					Metadata: map[string]interface{}{
						"build": true,
					},
				},
			},
		}

		_, err := os.Stat(filepath.Join(context.WorkingDir, "node_modules"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				nodeModulesRequirement := packit.BuildPlanRequirement{
					Name: "node_modules",
					Metadata: map[string]interface{}{
						"build": true,
					},
				}

				plan.Requires = append(plan.Requires, nodeModulesRequirement)
			} else {
				return packit.DetectResult{}, err
			}
		}

		return packit.DetectResult{Plan: plan}, nil
	}
}
