package step

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/export"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
)

type OutputExporter interface {
	ZipAndExportOutput(artifact, destinationZipPth, envKey string) error
	CopyAndSaveTestData(artifact, targetAddonPath, testName string) error
}

type outputExporter struct {
	exporter export.Exporter
}

func NewOutputExporter(cmdFactory command.Factory) OutputExporter {
	return outputExporter{
		exporter: export.NewDefaultExporter(cmdFactory),
	}
}

func (e outputExporter) ZipAndExportOutput(artifact, destinationZipPth, envKey string) error {
	return e.exporter.ExportOutputFilesZip(envKey, []string{artifact}, destinationZipPth)
}

func (e outputExporter) CopyAndSaveTestData(artifact, targetAddonPath, testName string) error {
	testName = replaceUnsupportedFilenameCharacters(testName)
	addonPerStepOutputDir := filepath.Join(targetAddonPath, testName)

	if err := copyDirectory(artifact, addonPerStepOutputDir); err != nil {
		return err
	}
	if err := saveBundleMetadata(addonPerStepOutputDir, testName); err != nil {
		return err
	}
	return nil
}

// Replaces characters '/' and ':', which are unsupported in filenames on macOS
func replaceUnsupportedFilenameCharacters(s string) string {
	s = strings.Replace(s, "/", "-", -1)
	s = strings.Replace(s, ":", "-", -1)
	return s
}

func copyDirectory(sourceBundle string, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory (%s): %w", targetDir, err)
	}

	cmd := command.NewFactory(env.NewRepository()).Create("cp", []string{"-a", sourceBundle, targetDir + "/"}, nil)
	if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
		return fmt.Errorf("copy failed: %w, output: %s", err, out)
	}

	return nil
}

func saveBundleMetadata(outputDir string, bundleName string) error {
	// Save test bundle metadata
	type testBundle struct {
		BundleName string `json:"test-name"`
	}
	bytes, err := json.Marshal(testBundle{
		BundleName: bundleName,
	})
	if err != nil {
		return fmt.Errorf("could not encode metadata: %w", err)
	}
	if err = os.WriteFile(filepath.Join(outputDir, "test-info.json"), bytes, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
