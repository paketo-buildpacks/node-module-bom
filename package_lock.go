package nodemodulebom

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit"
)

type PackageLock struct {
	Name            string                 `json:"name"`
	LockfileVersion int                    `json:"lockfileVersion"`
	Dependencies    map[string]interface{} `json:"dependencies"`
}

func NewPackageLock(workingDir string) (PackageLock, error) {
	file, err := os.Open(filepath.Join(workingDir, "package-lock.json"))
	if err != nil {
		return PackageLock{}, fmt.Errorf("failed to open package-lock.json: %w", err)
	}
	defer file.Close()

	var lockFileContent PackageLock
	err = json.NewDecoder(file).Decode(&lockFileContent)
	if err != nil {
		return PackageLock{}, fmt.Errorf("failed to decode package-lock: %w", err)
	}
	return lockFileContent, nil
}

// FindChecksum is a function that will read the package-lock.json if there is
// one, and retrieve the integrity (hash) for a specific dependency. It returns
// packit.BOMChecksum containing the algorithm and hash.
func (p PackageLock) FindChecksum(pkg string) (packit.BOMChecksum, error) {
	for name, dependency := range p.Dependencies {
		if name == pkg {
			dependencyMap := dependency.(map[string]interface{})
			integrity := dependencyMap["integrity"].(string)

			if strings.Contains(integrity, "-") {
				algAndHash := strings.Split(integrity, "-")
				hash, err := base64.StdEncoding.DecodeString(algAndHash[1])
				if err != nil {
					return packit.BOMChecksum{}, err
				}

				algorithm, err := packit.GetBOMChecksumAlgorithm(algAndHash[0])
				if err != nil {
					return packit.BOMChecksum{}, err
				}

				return packit.BOMChecksum{
					Algorithm: algorithm,
					Hash:      hex.EncodeToString(hash),
				}, nil
			}
			return packit.BOMChecksum{}, nil
		}
	}
	return packit.BOMChecksum{}, nil
}
