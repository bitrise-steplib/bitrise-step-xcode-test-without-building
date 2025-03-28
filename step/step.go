package step

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/xcodebuild"
	"github.com/kballard/go-shellquote"
)

const (
	testResultBundleKey       = "BITRISE_XCRESULT_PATH"
	zippedTestResultBundleKey = "BITRISE_XCRESULT_ZIP_PATH"
)

const (
	timeOutMessageIPhoneSimulator            = "iPhoneSimulator: Timed out waiting"
	timeOutMessageUITest                     = "Terminating app due to uncaught exception '_XCTestCaseInterruptionException'"
	earlyUnexpectedExit                      = "Early unexpected exit, operation never finished bootstrapping - no restart will be attempted"
	failureAttemptingToLaunch                = "Assertion Failure: <unknown>:0: UI Testing Failure - Failure attempting to launch <XCUIApplicationImpl:"
	failedToBackgroundTestRunner             = `Error Domain=IDETestOperationsObserverErrorDomain Code=12 "Failed to background test runner.`
	appStateIsStillNotRunning                = `App state is still not running active, state = XCApplicationStateNotRunning`
	appAccessibilityIsNotLoaded              = `UI Testing Failure - App accessibility isn't loaded`
	testRunnerFailedToInitializeForUITesting = `Test runner failed to initialize for UI testing`
	timedOutRegisteringForTestingEvent       = `Timed out registering for testing event accessibility notifications`
	testRunnerNeverBeganExecuting            = `Test runner never began executing tests after launching.`
	failedToOpenTestRunner                   = `Error Domain=FBSOpenApplicationServiceErrorDomain Code=1 "The request to open.*NSLocalizedFailureReason=The request was denied by service delegate \(SBMainWorkspace\)\.`
)

var testRunnerErrorPatterns = []string{
	timeOutMessageIPhoneSimulator,
	timeOutMessageUITest,
	earlyUnexpectedExit,
	failureAttemptingToLaunch,
	failedToBackgroundTestRunner,
	appStateIsStillNotRunning,
	appAccessibilityIsNotLoaded,
	testRunnerFailedToInitializeForUITesting,
	timedOutRegisteringForTestingEvent,
	testRunnerNeverBeganExecuting,
	failedToOpenTestRunner,
}

type Input struct {
	Xctestrun         string `env:"xctestrun,required"`
	Destination       string `env:"destination,required"`
	XcodebuildOptions string `env:"xcodebuild_options"`

	TestRepetitionMode             string `env:"test_repetition_mode,opt[none,until_failure,retry_on_failure,up_until_maximum_repetitions]"`
	MaximumTestRepetitions         int    `env:"maximum_test_repetitions,required"`
	RelaunchTestsForEachRepetition bool   `env:"relaunch_tests_for_each_repetition,opt[yes,no]"`

	DeployDir       string `env:"BITRISE_DEPLOY_DIR"`
	TestingAddonDir string `env:"BITRISE_TEST_RESULT_DIR"`

	OnlyTesting string `env:"only_testing"`
	SkipTesting string `env:"skip_testing"`
}

type Config struct {
	Xctestrun                      string
	Destination                    destination.Device
	XcodebuildOptions              []string
	TestRepetitionMode             string
	MaximumTestRepetitions         int
	RelaunchTestsForEachRepetition bool
	DeployDir                      string
	TestingAddonDir                string
	OnlyTesting                    []string
	SkipTesting                    []string
}

type Result struct {
	TestOutputDir   string
	DeployDir       string
	TestingAddonDir string
}

type XcodebuildTester struct {
	logger         log.Logger
	inputParser    stepconf.InputParser
	deviceFinder   destination.DeviceFinder
	pathChecker    pathutil.PathChecker
	xcodebuild     xcodebuild.Xcodebuild
	outputEnvStore env.Repository
	outputExporter OutputExporter
}

func NewXcodebuildTester(
	logger log.Logger,
	inputParser stepconf.InputParser,
	deviceFinder destination.DeviceFinder,
	pathChecker pathutil.PathChecker,
	xcodebuild xcodebuild.Xcodebuild,
	outputEnvStore env.Repository,
	outputExporter OutputExporter,
) XcodebuildTester {
	return XcodebuildTester{
		logger:         logger,
		inputParser:    inputParser,
		deviceFinder:   deviceFinder,
		pathChecker:    pathChecker,
		xcodebuild:     xcodebuild,
		outputEnvStore: outputEnvStore,
		outputExporter: outputExporter,
	}
}

func (s XcodebuildTester) ProcessConfig() (*Config, error) {
	var input Input
	if err := s.inputParser.Parse(&input); err != nil {
		return nil, err
	}

	stepconf.Print(input)

	xcodebuildOptions, err := shellquote.Split(input.XcodebuildOptions)
	if err != nil {
		return nil, fmt.Errorf("provided xcodebuild options (%s) are not valid CLI parameters: %w", input.XcodebuildOptions, err)
	}

	simulator, err := s.getSimulatorForDestination(input.Destination)
	if err != nil {
		return nil, err
	}

	s.logger.Infof("Simulator device:")
	s.logger.Printf("- name: %s, version: %s, UDID: %s, status: %s", simulator.Name, simulator.OS, simulator.ID, simulator.Status)

	onlyTesting, err := s.processTestConfiguration(input.OnlyTesting)
	if err != nil {
		return nil, err
	}

	skipTesting, err := s.processTestConfiguration(input.SkipTesting)
	if err != nil {
		return nil, err
	}

	return &Config{
		Xctestrun:                      input.Xctestrun,
		Destination:                    simulator,
		XcodebuildOptions:              xcodebuildOptions,
		TestRepetitionMode:             input.TestRepetitionMode,
		MaximumTestRepetitions:         input.MaximumTestRepetitions,
		RelaunchTestsForEachRepetition: input.RelaunchTestsForEachRepetition,
		DeployDir:                      input.DeployDir,
		TestingAddonDir:                input.TestingAddonDir,
		OnlyTesting:                    onlyTesting,
		SkipTesting:                    skipTesting,
	}, nil
}

func (s XcodebuildTester) Run(config Config) (*Result, error) {
	s.logger.Println()
	s.logger.Infof("Running tests:")

	result := &Result{
		DeployDir:       config.DeployDir,
		TestingAddonDir: config.TestingAddonDir,
	}

	runTests := func() (string, error) {
		return s.xcodebuild.TestWithoutBuilding(
			config.Xctestrun,
			config.OnlyTesting,
			config.SkipTesting,
			config.Destination,
			config.TestRepetitionMode,
			config.MaximumTestRepetitions,
			config.RelaunchTestsForEachRepetition,
			config.XcodebuildOptions...)
	}

	outputDir, err := runTests()
	if err != nil {
		var xcErr *xcodebuild.XcodebuildError
		if errors.As(err, &xcErr) {
			for _, errorPattern := range testRunnerErrorPatterns {
				if isStringFoundInOutput(errorPattern, xcErr.Log) {
					s.logger.Warnf("Automatic retry reason found in log: %s", errorPattern)
					outputDir, err = runTests()
				}
			}
		}
	}

	result.TestOutputDir = outputDir

	if err == nil {
		s.logger.TDonef("Passing tests")
	}

	return result, err
}

func (s XcodebuildTester) ExportOutputs(result Result) error {
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

func (s XcodebuildTester) getSimulatorForDestination(destinationSpecifier string) (destination.Device, error) {
	simulatorDestination, err := destination.NewSimulator(destinationSpecifier)
	if err != nil {
		return destination.Device{}, fmt.Errorf("invalid destination specifier (%s): %w", destinationSpecifier, err)
	}

	s.logger.Println()
	device, err := s.deviceFinder.FindDevice(*simulatorDestination)
	if err != nil {
		return destination.Device{}, fmt.Errorf("simulator UDID lookup failed: %w", err)
	}

	return device, nil
}

func (s XcodebuildTester) processTestConfiguration(input string) ([]string, error) {
	if input == "" {
		return nil, nil
	}

	exists, err := s.pathChecker.IsPathExists(input)
	if err != nil {
		return nil, err
	}

	var contents string

	if exists {
		bytes, err := os.ReadFile(input)
		if err != nil {
			return nil, err
		}

		contents = string(bytes)
	} else {
		contents = input
	}

	identifiers := strings.Split(contents, "\n")

	return removeEmptyLines(identifiers), nil
}

func isStringFoundInOutput(searchStr, outputToSearchIn string) bool {
	r := regexp.MustCompile("(?i)" + searchStr)
	return r.MatchString(outputToSearchIn)
}

func removeEmptyLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}
