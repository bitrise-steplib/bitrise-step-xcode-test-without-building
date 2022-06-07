package step

import (
	"testing"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/mocks"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/xcodebuild"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_GivenStep_WhenProcessConfig_ThenSplitsAdditionalOptions(t *testing.T) {
	// Given
	step, testingMocks := createStepAndMocks()

	inputs := map[string]string{
		"xctestrun":                          "my_test.xctestrun",
		"destination":                        "platform=iOS Simulator,name=iPhone 8 Plus,OS=latest",
		"test_repetition_mode":               "none",
		"maximum_test_repetitions":           "3",
		"relaunch_tests_for_each_repetition": "no",
		"xcodebuild_options":                 "-parallel-testing-enabled YES",
	}
	for key, value := range inputs {
		testingMocks.envRepository.On("Get", key).Return(value)
	}
	testingMocks.envRepository.On("Get", mock.Anything).Return("")

	// When
	config, err := step.ProcessConfig()

	// Then
	require.NoError(t, err)
	require.Equal(t, []string{"-parallel-testing-enabled", "YES"}, config.XcodebuildOptions)
}

func Test_GivenStep_WhenXcodebuildFailsOnAutomaticRetryReason_ThenXcodebuildCommandRetried(t *testing.T) {
	// Given
	step, testingMocks := createStepAndMocks()

	testingMocks.xcodebuild.On("TestWithoutBuilding", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", &xcodebuild.XcodebuildError{Log: "Test runner never began executing tests after launching."})
	testingMocks.logger.On("Println").Return()
	testingMocks.logger.On("Infof", mock.Anything).Return()
	testingMocks.logger.On("Warnf", mock.Anything, mock.Anything).Return()

	config := Config{
		Xctestrun:                      "",
		Destination:                    "",
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
	step, testingMocks := createStepAndMocks()

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
	step, testingMocks := createStepAndMocks()

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
	xcodebuild     *mocks.Xcodebuild
	outputExporter *mocks.OutputExporter
}

func createStepAndMocks() (XcodebuildTest, testingMocks) {
	envRepository := new(mocks.Repository)
	inputParser := stepconf.NewInputParser(envRepository)
	logger := new(mocks.Logger)
	xcodebuild := new(mocks.Xcodebuild)
	outputExporter := new(mocks.OutputExporter)
	step := NewXcodebuildTest(logger, inputParser, xcodebuild, envRepository, outputExporter)

	mocks := testingMocks{
		envRepository:  envRepository,
		inputParser:    inputParser,
		logger:         logger,
		xcodebuild:     xcodebuild,
		outputExporter: outputExporter,
	}

	return step, mocks
}
