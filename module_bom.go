package nodemodulebom

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(execution pexec.Execution) error
}

type ModuleSBOM struct {
	executable Executable
	logger     scribe.Emitter
}

func NewModuleSBOM(executable Executable, logger scribe.Emitter) ModuleSBOM {
	return ModuleSBOM{
		executable: executable,
		logger:     logger,
	}
}

func (m ModuleSBOM) Generate(workingDir, layersDir, layerName string) error {
	buffer := bytes.NewBuffer(nil)
	args := []string{"packages", fmt.Sprintf("dir:%s", workingDir), "--output", "json", "--file", filepath.Join(layersDir, fmt.Sprintf("%s.sbom.syft.json", layerName))}
	m.logger.Subprocess("Running 'syft %s'", strings.Join(args, " "))
	err := m.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		m.logger.Detail(buffer.String())
		return err
	}
	return nil
}
