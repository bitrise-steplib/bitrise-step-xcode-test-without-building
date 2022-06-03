package xcodebuild

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

type Xcodebuild interface {
	TestWithoutBuilding(xctestrun, destination string, options ...string) (string, error)
}

type xcodebuild struct {
	logger         log.Logger
	commandFactory command.Factory
	pathProvider   pathutil.PathProvider
	pathChecker    pathutil.PathChecker
}

func New(logger log.Logger, commandFactory command.Factory, pathProvider pathutil.PathProvider, pathChecker pathutil.PathChecker) Xcodebuild {
	return xcodebuild{
		commandFactory: commandFactory,
		logger:         logger,
		pathProvider:   pathProvider,
		pathChecker:    pathChecker,
	}
}

func (x xcodebuild) TestWithoutBuilding(xctestrun, destination string, opts ...string) (string, error) {
	outputDir, err := x.createTestOutputDir(xctestrun)
	if err != nil {
		return "", err
	}

	options := []string{"test-without-building", "-xctestrun", xctestrun, "-destination", destination, "-resultBundlePath", outputDir}
	options = append(options, opts...)

	cmd := x.commandFactory.Create("xcodebuild", options, &command.Opts{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})

	x.logger.TDonef(cmd.PrintableCommandArgs())
	err = cmd.Run()

	if exist, _ := x.pathChecker.IsPathExists(outputDir); !exist {
		outputDir = ""
	}
	return outputDir, err
}

func (x xcodebuild) createTestOutputDir(xctestrun string) (string, error) {
	tempDir, err := x.pathProvider.CreateTempDir("XCUITestOutput")
	if err != nil {
		return "", fmt.Errorf("could not create test output temporary directory: %w", err)
	}

	fileName := strings.TrimSuffix(filepath.Base(xctestrun), filepath.Ext(xctestrun))
	return path.Join(tempDir, fmt.Sprintf("Test-%s.xcresult", fileName)), nil
}
