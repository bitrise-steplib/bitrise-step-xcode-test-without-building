package step

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/xcodebuild"
	"github.com/kballard/go-shellquote"
)

const (
	testResultBundleKey       = "BITRISE_XCRESULT_PATH"
	zippedTestResultBundleKey = "BITRISE_XCRESULT_ZIP_PATH"
)

type Input struct {
	Xctestrun         string `env:"xctestrun,required"`
	Destination       string `env:"destination,required"`
	XcodebuildOptions string `env:"xcodebuild_options"`
	DeployDir         string `env:"BITRISE_DEPLOY_DIR"`
	TestingAddonDir   string `env:"BITRISE_TEST_RESULT_DIR"`
}

type Config struct {
	Xctestrun         string
	Destination       string
	XcodebuildOptions []string
	DeployDir         string
	TestingAddonDir   string
}

type Result struct {
	TestOutputDir   string
	DeployDir       string
	TestingAddonDir string
}

type Step struct {
	logger         log.Logger
	inputParser    stepconf.InputParser
	xcodebuild     xcodebuild.Xcodebuild
	outputEnvStore env.Repository
	outputExporter OutputExporter
}

func New(logger log.Logger, inputParser stepconf.InputParser, xcodebuild xcodebuild.Xcodebuild, outputEnvStore env.Repository, outputExporter OutputExporter) Step {
	return Step{
		logger:         logger,
		inputParser:    inputParser,
		xcodebuild:     xcodebuild,
		outputEnvStore: outputEnvStore,
		outputExporter: outputExporter,
	}
}

func (s Step) ProcessConfig() (*Config, error) {
	var input Input
	if err := s.inputParser.Parse(&input); err != nil {
		return nil, err
	}

	stepconf.Print(input)

	xcodebuildOptions, err := shellquote.Split(input.XcodebuildOptions)
	if err != nil {
		return nil, fmt.Errorf("provided xcodebuild options (%s) are not valid CLI parameters: %w", input.XcodebuildOptions, err)
	}

	return &Config{
		Xctestrun:         input.Xctestrun,
		Destination:       input.Destination,
		XcodebuildOptions: xcodebuildOptions,
		DeployDir:         input.DeployDir,
		TestingAddonDir:   input.TestingAddonDir,
	}, nil
}

func (s Step) InstallDependencies() error {
	return nil
}

func (s Step) Run(config Config) (*Result, error) {
	s.logger.Println()
	s.logger.Infof("Running tests:")

	result := &Result{
		DeployDir:       config.DeployDir,
		TestingAddonDir: config.TestingAddonDir,
	}

	outputDir, err := s.xcodebuild.TestWithoutBuilding(config.Xctestrun, config.Destination, config.XcodebuildOptions...)
	result.TestOutputDir = outputDir

	if err != nil {
		var exerr *exec.ExitError
		if errors.As(err, &exerr) {
			return result, fmt.Errorf("failing tests (exist status %v)", exerr.ExitCode())
		} else {
			return result, fmt.Errorf("test execute failed: %w", err)
		}
	}

	s.logger.TDonef("Passing tests")
	return result, err
}

func (s Step) ExportOutputs(result Result) error {
	s.logger.Println()
	s.logger.Infof("Exporting outputs:")

	if result.TestOutputDir != "" {
		if err := s.outputEnvStore.Set(testResultBundleKey, result.TestOutputDir); err != nil {
			s.logger.Warnf("Failed to export: %s: %s", testResultBundleKey, err)
		} else {
			s.logger.Donef("%s: %s", testResultBundleKey, result.TestOutputDir)
		}

		if result.DeployDir != "" {
			xcresultZipPath := filepath.Join(result.DeployDir, filepath.Base(result.TestOutputDir)+".zip")
			if err := s.outputExporter.ZipAndExportOutput(result.TestOutputDir, xcresultZipPath, zippedTestResultBundleKey); err != nil {
				s.logger.Warnf("Failed to export: %s: %s", zippedTestResultBundleKey, err)
			} else {
				s.logger.Donef("%s: %s", zippedTestResultBundleKey, xcresultZipPath)
			}
		}

		if result.TestingAddonDir != "" {
			testName := strings.TrimSuffix(filepath.Base(result.TestOutputDir), filepath.Ext(result.TestOutputDir))

			if err := s.outputExporter.CopyAndSaveTestData(result.TestOutputDir, result.TestingAddonDir, testName); err != nil {
				s.logger.Warnf("Testing addon export failed: %s", err)
			} else {
				s.logger.Donef("Test result bundle moved to the testing addon dir: %s", result.TestingAddonDir)
			}
		}
	}
	return nil
}
