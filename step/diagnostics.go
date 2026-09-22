package step

const (
	diagnosticsProjectSetting = "project_setting"
	diagnosticsNever          = "never"
)

// minimumXcodeMajorWithDiagnosticsOption is the first Xcode version that understands
// -collect-test-diagnostics and collects Simulator diagnostics itself after a failing test run.
const minimumXcodeMajorWithDiagnosticsOption = 26

func xcodebuildDiagnosticsOverride(condition string, xcodeMajorVersion int64, additionalOptions []string) string {
	if xcodeMajorVersion < minimumXcodeMajorWithDiagnosticsOption { // no such option yet
		return ""
	}

	for _, option := range additionalOptions {
		if option == "-collect-test-diagnostics" { // user's option wins
			return ""
		}
	}

	switch condition {
	case diagnosticsProjectSetting: // leave it to the test plan
		return ""
	case diagnosticsNever:
		return "never"
	default:
		return "on-failure"
	}
}
