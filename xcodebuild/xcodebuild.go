package xcodebuild

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

const (
	TestRepetitionNone           = "none"
	TestRepetitionUntilFailure   = "until_failure"
	TestRepetitionRetryOnFailure = "retry_on_failure"
)

type Xcodebuild interface {
	TestWithoutBuilding(xctestrun, destination, testRepetitionMode string, maximumTestRepetitions int, relaunchTestsForEachRepetition bool, options ...string) (string, error)
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

func (x xcodebuild) TestWithoutBuilding(xctestrun, destination, testRepetitionMode string, maximumTestRepetitions int, relaunchTestsForEachRepetition bool, opts ...string) (string, error) {
	logFile, err := x.createXcodebuildLogFile()
	if err != nil {
		return "", err
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			x.logger.Warnf("Failed to open xcodebuild log file: %s", err)
		}
	}()

	outputWriter := io.MultiWriter(os.Stdout, logFile)

	outputDir, err := x.createTestOutputDir(xctestrun)
	if err != nil {
		return "", err
	}

	options := createXcodebuildOptions(xctestrun, destination, testRepetitionMode, maximumTestRepetitions, relaunchTestsForEachRepetition, outputDir)
	cmd := x.commandFactory.Create("xcodebuild", options, &command.Opts{
		Stdout: outputWriter,
		Stderr: outputWriter,
	})

	x.logger.TDonef(cmd.PrintableCommandArgs())
	xcodebuildErr := cmd.Run()

	return x.handleError(xcodebuildErr, outputDir, logFile)
}

func (x xcodebuild) createXcodebuildLogFile() (*os.File, error) {
	tempDir, err := x.pathProvider.CreateTempDir("xcodebuild")
	if err != nil {
		return nil, err
	}

	return os.Create(path.Join(tempDir, "test-without-building.log"))
}

func (x xcodebuild) createTestOutputDir(xctestrun string) (string, error) {
	tempDir, err := x.pathProvider.CreateTempDir("TestOutput")
	if err != nil {
		return "", err
	}

	fileName := strings.TrimSuffix(filepath.Base(xctestrun), filepath.Ext(xctestrun))
	return path.Join(tempDir, fmt.Sprintf("Test-%s.xcresult", fileName)), nil
}

func (x xcodebuild) handleError(xcodebuildErr error, outputDir string, logFile *os.File) (string, error) {
	empty, err := isDirEmpty(outputDir)
	if err != nil {
		x.logger.Warnf("Failed to check if test result bundle is empty: %s", err)
	}
	if empty {
		outputDir = ""
	}

	if xcodebuildErr != nil {
		var exerr *exec.ExitError
		if errors.As(xcodebuildErr, &exerr) {
			log, err := io.ReadAll(logFile)
			if err != nil {
				x.logger.Warnf("Failed to open xcodebuild log file: %s", err)
			}

			return outputDir, &XcodebuildError{
				Reason: fmt.Sprintf("failing tests (exist status %v)", exerr.ExitCode()),
				Err:    xcodebuildErr,
				Log:    string(log),
			}
		}

		return outputDir, fmt.Errorf("test execute failed: %w", xcodebuildErr)
	}

	return outputDir, nil
}

func createXcodebuildOptions(xctestrun, destination, testRepetitionMode string, maximumTestRepetitions int, relaunchTestsForEachRepetition bool, outputDir string, opts ...string) []string {
	options := []string{"test-without-building", "-xctestrun", xctestrun, "-destination", destination, "-resultBundlePath", outputDir}
	switch testRepetitionMode {
	case TestRepetitionUntilFailure:
		options = append(options, "-run-tests-until-failure")
	case TestRepetitionRetryOnFailure:
		options = append(options, "-retry-tests-on-failure")
	}
	if testRepetitionMode != TestRepetitionNone {
		options = append(options, "-test-iterations", strconv.Itoa(maximumTestRepetitions))
	}
	if relaunchTestsForEachRepetition {
		options = append(options, "-test-repetition-relaunch-enabled", "YES")
	}
	return append(options, opts...)
}

func isDirEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)

	if err == io.EOF {
		return true, nil
	}
	return false, err
}
