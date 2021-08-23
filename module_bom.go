package nodemodulebom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(execution pexec.Execution) error
}

type ModuleBOM struct {
	executable Executable
	logger     scribe.Emitter
}

func NewModuleBOM(executable Executable, logger scribe.Emitter) ModuleBOM {
	return ModuleBOM{
		executable: executable,
		logger:     logger,
	}
}

func (m ModuleBOM) Generate(workingDir string) ([]packit.BOMEntry, error) {

	buffer := bytes.NewBuffer(nil)
	args := []string{"-o", "bom.json"}
	m.logger.Subprocess("Running 'cyclonedx-bom %s'", strings.Join(args, " "))
	err := m.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		m.logger.Detail(buffer.String())
		return nil, fmt.Errorf("failed to run cyclonedx-bom: %w", err)
	}

	file, err := os.Open(filepath.Join(workingDir, "bom.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to open bom.json: %w", err)
	}
	defer file.Close()

	var bom struct {
		Components []struct {
			Name    string `json:"name"`
			PURL    string `json:"purl"`
			Version string `json:"version"`
			Hashes  []struct {
				Algorithm string `json:"alg"`
				Content   string `json:"content"`
			} `json:"hashes"`
			Licenses []struct {
				License struct {
					ID string `json:"id"`
				} `json:"license"`
			} `json:"licenses"`
		} `json:"components"`
	}

	err = json.NewDecoder(file).Decode(&bom)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bom.json: %w", err)
	}

	var entries []packit.BOMEntry
	for _, entry := range bom.Components {
		packitEntry := packit.BOMEntry{
			Name: entry.Name,
			Metadata: map[string]interface{}{
				"version": entry.Version,
				"purl":    entry.PURL,
			},
		}

		if len(entry.Hashes) > 0 {
			packitEntry.Metadata["checksum"] = map[string]string{
				"algorithm": entry.Hashes[0].Algorithm,
				"hash":      entry.Hashes[0].Content,
			}
		}

		var licenses []string
		for _, license := range entry.Licenses {
			licenses = append(licenses, license.License.ID)
		}
		packitEntry.Metadata["licenses"] = licenses
		entries = append(entries, packitEntry)
	}

	err = os.Remove(filepath.Join(workingDir, "bom.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to remove bom.json: %w", err)
	}

	return entries, nil
}
