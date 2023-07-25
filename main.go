package main

import (
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-steputils/v2/stepenv"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/step"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/xcodebuild"
)

func main() {
	os.Exit(run())
}

func run() int {
	exitCode := 0

	logger := log.NewLogger()
	xcodebuildTester := createXcodebuildTester(logger)

	config, err := xcodebuildTester.ProcessConfig()
	if err != nil {
		logger.Errorf(err.Error())
		exitCode = 1
		return exitCode
	}

	result, err := xcodebuildTester.Run(*config)
	if err != nil {
		logger.TErrorf(err.Error())
		exitCode = 1
	}

	if err = xcodebuildTester.ExportOutputs(*result); err != nil {
		logger.Errorf(err.Error())
		exitCode = 1
	}

	return exitCode
}

func createXcodebuildTester(logger log.Logger) step.XcodebuildTester {
	osEnvs := env.NewRepository()
	inputParser := stepconf.NewInputParser(osEnvs)
	outputEnvStore := stepenv.NewRepository(osEnvs)
	commandFactory := command.NewFactory(osEnvs)
	pathProvider := pathutil.NewPathProvider()
	pathChecker := pathutil.NewPathChecker()
	xcodeversionProvider := xcodeversion.NewXcodeVersionProvider(commandFactory)
	xcodeVersion, err := xcodeversionProvider.GetVersion()
	if err != nil {
		logger.Errorf("failed to read Xcode version: %s", err) // not a fatal error, continuing
	}
	deviceFinder := destination.NewDeviceFinder(logger, commandFactory, xcodeVersion)
	xcbuild := xcodebuild.New(logger, commandFactory, pathProvider, pathChecker)
	outputExporter := step.NewOutputExporter()

	return step.NewXcodebuildTester(logger, inputParser, deviceFinder, xcbuild, outputEnvStore, outputExporter)
}
