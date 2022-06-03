package step

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/output"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
)

type OutputExporter interface {
	ZipAndExportOutput(artifact string, destinationZipPth, envKey string) error
	CopyAndSaveTestData(sourceTestOutputDir, targetAddonPath, targetAddonBundleName string) error
}

type outputExporter struct {
}

func NewOutputExporter() OutputExporter {
	return outputExporter{}
}

func (e outputExporter) ZipAndExportOutput(artifact string, destinationZipPth, envKey string) error {
	return output.ZipAndExportOutput([]string{artifact}, destinationZipPth, envKey)
}

func (e outputExporter) CopyAndSaveTestData(sourceTestOutputDir, targetAddonPath, targetAddonBundleName string) error {
	targetAddonBundleName = replaceUnsupportedFilenameCharacters(targetAddonBundleName)
	addonPerStepOutputDir := filepath.Join(targetAddonPath, targetAddonBundleName)

	if err := copyDirectory(sourceTestOutputDir, addonPerStepOutputDir); err != nil {
		return err
	}
	if err := saveBundleMetadata(addonPerStepOutputDir, targetAddonBundleName); err != nil {
		return err
	}
	return nil
}

// Replaces characters '/' and ':', which are unsupported in filnenames on macOS
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
	if err = ioutil.WriteFile(filepath.Join(outputDir, "test-info.json"), bytes, 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
