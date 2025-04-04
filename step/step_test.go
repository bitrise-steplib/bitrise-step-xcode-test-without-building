package step

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/mocks"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/xcodebuild"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GivenStep_WhenProcessConfig_ThenSplitsAdditionalOptions(t *testing.T) {
	// Given
	step, testingMocks := createStepAndMocks(t)

	onlyTesting := []string{
		"target1",
		"target2/testClass1",
		"target3/testClass1/testFunction",
	}
	skipTesting := []string{
		"target4",
		"target5/testClass1",
		"target6/testClass1/testFunction",
	}

	path := filepath.Join(t.TempDir(), "skip_testing.txt")
	err := os.WriteFile(path, []byte(strings.Join(skipTesting, "\n")), 0644)
	require.NoError(t, err)

	inputs := map[string]string{
		"xctestrun":                          "my_test.xctestrun",
		"destination":                        "platform=iOS Simulator,name=iPhone 8 Plus,OS=latest",
		"test_repetition_mode":               "none",
		"maximum_test_repetitions":           "3",
		"relaunch_tests_for_each_repetition": "no",
		"xcodebuild_options":                 "-parallel-testing-enabled YES",
		"only_testing":                       strings.Join(onlyTesting, "\n"),
		"skip_testing":                       path,
	}
	for key, value := range inputs {
		testingMocks.envRepository.On("Get", key).Return(value)
	}

	testingMocks.envRepository.On("Get", mock.Anything).Return("")
	testingMocks.deviceFinder.On("FindDevice", mock.Anything, mock.Anything).Return(destination.Device{
		ID: "test-UDID",
	}, nil)

	// When
	config, err := step.ProcessConfig()

	// Then
	require.NoError(t, err)
	require.Equal(t, []string{"-parallel-testing-enabled", "YES"}, config.XcodebuildOptions)
	require.Equal(t, onlyTesting, config.OnlyTesting)
	require.Equal(t, skipTesting, config.SkipTesting)
}

func Test_GivenStep_WhenXcodebuildFailsOnAutomaticRetryReason_ThenXcodebuildCommandRetried(t *testing.T) {
	// Given
	step, testingMocks := createStepAndMocks(t)

	testingMocks.xcodebuild.On("TestWithoutBuilding", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", &xcodebuild.XcodebuildError{Log: "Test runner never began executing tests after launching."})
	testingMocks.logger.On("Println").Return()
	testingMocks.logger.On("Infof", mock.Anything).Return()
	testingMocks.logger.On("Warnf", mock.Anything, mock.Anything).Return()

	config := Config{
		Xctestrun:                      "",
		Destination:                    destination.Device{ID: "test-UDID"},
		XcodebuildOptions:              nil,
		TestRepetitionMode:             "",
		MaximumTestRepetitions:         0,
		RelaunchTestsForEachRepetition: false,
		DeployDir:                      "",
		TestingAddonDir:                "",
	}

	// When
	_, err := step.Run(config)

	// Then
	require.Error(t, err)
	testingMocks.xcodebuild.AssertNumberOfCalls(t, "TestWithoutBuilding", 2)
}

func Test_GivenDeployDir_WhenStepExportsOutputs_ThenTestResultMovedToDeployDir(t *testing.T) {
	// Given
	step, testingMocks := createStepAndMocks(t)

	testingMocks.logger.On("Println").Return()
	testingMocks.logger.On("Infof", mock.Anything).Return()
	testingMocks.logger.On("Donef", mock.Anything, mock.Anything, mock.Anything).Return()
	testingMocks.envRepository.On("Set", mock.Anything, mock.Anything).Return(nil)

	result := Result{
		TestOutputDir: "my_test.xcresult",
		DeployDir:     "deploy_dir",
	}

	testingMocks.outputExporter.On("ZipAndExportOutput", result.TestOutputDir, mock.Anything, mock.Anything).Return(nil)

	// When
	err := step.ExportOutputs(result)

	// Then
	require.NoError(t, err)
	testingMocks.outputExporter.AssertExpectations(t)
}

func Test_GivenTestingAddonDir_WhenStepExportsOutputs_ThenTestResultMovedToTestingAddonDir(t *testing.T) {
	// Given
	step, testingMocks := createStepAndMocks(t)

	testingMocks.logger.On("Println").Return()
	testingMocks.logger.On("Infof", mock.Anything).Return()
	testingMocks.logger.On("Donef", mock.Anything, mock.Anything, mock.Anything).Return()
	testingMocks.envRepository.On("Set", mock.Anything, mock.Anything).Return(nil)

	result := Result{
		TestOutputDir:   "my_test.xcresult",
		TestingAddonDir: "testing_addon_dir",
	}

	testingMocks.outputExporter.On("CopyAndSaveTestData", result.TestOutputDir, mock.Anything, mock.Anything).Return(nil)

	// When
	err := step.ExportOutputs(result)

	// Then
	require.NoError(t, err)
	testingMocks.outputExporter.AssertExpectations(t)
}

type testingMocks struct {
	envRepository  *mocks.Repository
	inputParser    stepconf.InputParser
	logger         *mocks.Logger
	deviceFinder   *mocks.DeviceFinder
	xcodebuild     *mocks.Xcodebuild
	outputExporter *mocks.OutputExporter
}

func createStepAndMocks(t *testing.T) (XcodebuildTester, testingMocks) {
	envRepository := new(mocks.Repository)
	inputParser := stepconf.NewInputParser(envRepository)
	logger := new(mocks.Logger)
	deviceFinder := mocks.NewDeviceFinder(t)
	xcbuild := new(mocks.Xcodebuild)
	outputExporter := new(mocks.OutputExporter)
	pathChecker := pathutil.NewPathChecker()
	step := NewXcodebuildTester(log.NewLogger(), inputParser, deviceFinder, pathChecker, xcbuild, envRepository, outputExporter)

	m := testingMocks{
		envRepository:  envRepository,
		inputParser:    inputParser,
		logger:         logger,
		deviceFinder:   deviceFinder,
		xcodebuild:     xcbuild,
		outputExporter: outputExporter,
	}

	return step, m
}
