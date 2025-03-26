package xcodebuild

import (
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-utils/v2/pathutil"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/mocks"
)

func TestTestConfiguration(t *testing.T) {
	commandMock := new(mocks.Command)
	commandMock.On("PrintableCommandArgs").Return("")
	commandMock.On("Run").Return(nil)

	params := []string{"test-without-building", "-xctestrun", "test.xctestrun", "-destination", "id=test-UDID", "-resultBundlePath", "/test/path/Test-test.xcresult", "-only-testing:target1", "-only-testing:target2/testClass1", "-only-testing:target3/testClass1/testFunction", "-skip-testing:target4", "-skip-testing:target5/testClass1", "-skip-testing:target6/testClass1/testFunction"}

	factoryMock := new(mocks.Factory)
	factoryMock.On("Create", "xcodebuild", params, mock.Anything).Return(commandMock, nil).Once()

	pathProviderMock := new(mocks.PathProvider)
	pathProviderMock.On("CreateTempDir", "xcodebuild").Return(os.TempDir(), nil).Once()
	pathProviderMock.On("CreateTempDir", "TestOutput").Return("/test/path", nil).Once()

	xcbuild := New(log.NewLogger(), factoryMock, pathProviderMock, pathutil.NewPathChecker())
	device := destination.Device{ID: "test-UDID"}
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

	_, err := xcbuild.TestWithoutBuilding("test.xctestrun", onlyTesting, skipTesting, device, "none", 0, false)
	require.NoError(t, err)

	pathProviderMock.AssertExpectations(t)
	commandMock.AssertExpectations(t)
	factoryMock.AssertExpectations(t)
}
