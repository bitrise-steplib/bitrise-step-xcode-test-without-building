package main

import (
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-steputils/v2/stepenv"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/step"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/xcodebuild"
)

func main() {
	os.Exit(run())
}

func run() int {
	exitCode := 0

	logger := log.NewLogger()
	step := createStep(logger)

	config, err := step.ProcessConfig()
	if err != nil {
		logger.Errorf(err.Error())
		exitCode = 1
		return exitCode
	}

	result, err := step.Run(*config)
	if err != nil {
		logger.TErrorf(err.Error())
		exitCode = 1
	}

	if err = step.ExportOutputs(*result); err != nil {
		logger.Errorf(err.Error())
		exitCode = 1
	}

	return exitCode
}

func createStep(logger log.Logger) step.Step {
	osEnvs := env.NewRepository()
	inputParser := stepconf.NewInputParser(osEnvs)
	outputEnvStore := stepenv.NewRepository(osEnvs)
	commandFactory := command.NewFactory(osEnvs)
	pathProvider := pathutil.NewPathProvider()
	pathChecker := pathutil.NewPathChecker()
	xcodebuild := xcodebuild.New(logger, commandFactory, pathProvider, pathChecker)
	outputExporter := step.NewOutputExporter()

	return step.New(logger, inputParser, xcodebuild, outputEnvStore, outputExporter)
}
