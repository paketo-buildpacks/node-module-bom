package nodemodulebom

import (
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

		return packit.DetectResult{Plan: plan}, nil
	}
}
