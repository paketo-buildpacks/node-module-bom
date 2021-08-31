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

type packageLockStruct struct {
	Name            string                 `json:"name"`
	LockfileVersion int                    `json:"lockfileVersion"`
	Dependencies    map[string]interface{} `json:"dependencies"`
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

	var packageLock packageLockStruct
	var entries []packit.BOMEntry
	for _, entry := range bom.Components {
		packitEntry := packit.BOMEntry{
			Name: entry.Name,
			Metadata: packit.BOMMetadata{
				Version: entry.Version,
				PURL:    entry.PURL,
			},
		}

		if len(entry.Hashes) > 0 {
			algorithm, err := packit.GetBOMChecksumAlgorithm(entry.Hashes[0].Algorithm)
			if err != nil {
				return nil, err
			}
			packitEntry.Metadata.Checksum = packit.BOMChecksum{
				Algorithm: algorithm,
				Hash:      entry.Hashes[0].Content,
			}
		} else {
			// if the cyclonedx-bom tool doesn't find a hash (Node Engine v15+)
			// look up the integrity field for the package from the `package-lock.json`
			if packageLock.Name == "" {
				packageLock, err = getLockFile(workingDir)
			}

			// if the package-lock.json is not retrievable, do not error out
			// just skip trying to get checksums
			if err == nil {
				alg, hash := retrieveIntegrityFromLockfile(packageLock, entry.Name)
				packitEntry.Metadata["checksum"] = map[string]string{
					"algorithm": alg,
					"hash":      hash,
				}
			}
		}

		var licenses []string
		for _, license := range entry.Licenses {
			licenses = append(licenses, license.License.ID)
		}
		packitEntry.Metadata.Licenses = licenses
		entries = append(entries, packitEntry)
	}

	err = os.Remove(filepath.Join(workingDir, "bom.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to remove bom.json: %w", err)
	}

	return entries, nil
}

func getLockFile(workingDir string) (packageLockStruct, error) {
	file, err := os.Open(filepath.Join(workingDir, "package-lock.json"))
	if err != nil {
		return packageLockStruct{}, fmt.Errorf("failed to open package-lock.json: %w", err)
	}
	defer file.Close()

	var lockFile packageLockStruct
	err = json.NewDecoder(file).Decode(&lockFile)
	if err != nil {
		return packageLockStruct{}, fmt.Errorf("failed to decode package-lock: %w", err)
	}
	return lockFile, nil
}

// retrieveIntegrityFromLockfile is a function that will read the
// package-lock.json if there is one, and retrieve the integrity (hash) for a
// specific dependency. It returns the hash algorithm and hash itself.
func retrieveIntegrityFromLockfile(packageLock packageLockStruct, pkg string) (string, string) {
	for name, dependency := range packageLock.Dependencies {
		if name == pkg {
			dependencyMap := dependency.(map[string]interface{})
			integrity := dependencyMap["integrity"].(string)

			if strings.Contains(integrity, "-") {
				algAndHash := strings.Split(integrity, "-")
				return algAndHash[0], algAndHash[1]
			}
			return "", ""
		}
	}
	return "", ""
}
