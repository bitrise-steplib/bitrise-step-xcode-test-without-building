package step

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_xcodebuildDiagnosticsOverride(t *testing.T) {
	tests := []struct {
		name              string
		condition         string
		xcodeMajorVersion int64
		additionalOptions []string
		want              string
	}{
		{name: "Xcode 27, never", condition: "never", xcodeMajorVersion: 27, want: "never"},
		{name: "Xcode 26, never", condition: "never", xcodeMajorVersion: 26, want: "never"},
		{name: "Xcode 27, on_failure", condition: "on_failure", xcodeMajorVersion: 27, want: "on-failure"},
		{name: "Xcode 27, project_setting - option not passed", condition: "project_setting", xcodeMajorVersion: 27, want: ""},
		// The option does not exist before Xcode 26; passing it would be a usage error.
		{name: "Xcode 16, never - option not passed", condition: "never", xcodeMajorVersion: 16, want: ""},
		// main.go leaves the major at 0 when the Xcode version cannot be read.
		{name: "unknown Xcode - option not passed", condition: "never", xcodeMajorVersion: 0, want: ""},
		{name: "user set the option explicitly - theirs wins", condition: "never", xcodeMajorVersion: 27, additionalOptions: []string{"-collect-test-diagnostics", "on-failure"}, want: ""},
		// xcodebuild silently drops the -option=value form, so it must not count as an override.
		{name: "user wrote the option with = syntax - Step still passes its own", condition: "never", xcodeMajorVersion: 27, additionalOptions: []string{"-collect-test-diagnostics=on-failure"}, want: "never"},
		{name: "unrelated additional options are ignored", condition: "never", xcodeMajorVersion: 27, additionalOptions: []string{"-parallel-testing-enabled", "YES"}, want: "never"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, xcodebuildDiagnosticsOverride(tt.condition, tt.xcodeMajorVersion, tt.additionalOptions))
		})
	}
}
