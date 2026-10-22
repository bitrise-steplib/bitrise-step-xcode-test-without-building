package step

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult3/model3"
	"github.com/bitrise-steplib/bitrise-step-test-selection/selector"
	"github.com/stretchr/testify/require"
)

const xctestrunV2 = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>TestConfigurations</key>
	<array>
		<dict>
			<key>Name</key>
			<string>Test Scheme Action</string>
			<key>TestTargets</key>
			<array>
				<dict><key>BlueprintName</key><string>CoreTests</string></dict>
				<dict><key>BlueprintName</key><string>FeatureTests</string></dict>
			</array>
		</dict>
	</array>
	<key>__xctestrun_metadata__</key>
	<dict><key>FormatVersion</key><integer>2</integer></dict>
</dict>
</plist>`

const xctestrunV1 = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>AppTests</key>
	<dict><key>TestHostPath</key><string>__TESTROOT__/App.app</string></dict>
	<key>AppUITests</key>
	<dict><key>IsUITestBundle</key><true/></dict>
	<key>__xctestrun_metadata__</key>
	<dict><key>FormatVersion</key><integer>1</integer></dict>
</dict>
</plist>`

func TestXctestrunTargets(t *testing.T) {
	dir := t.TempDir()
	for content, want := range map[string][]string{
		xctestrunV2: {"CoreTests", "FeatureTests"},
		xctestrunV1: {"AppTests", "AppUITests"},
	} {
		p := filepath.Join(dir, "x.xctestrun")
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		got, err := xctestrunTargets(p)
		require.NoError(t, err)
		sort.Strings(got)
		require.Equal(t, want, got)
	}
	_, err := xctestrunTargets(filepath.Join(dir, "missing.xctestrun"))
	require.Error(t, err)
}

func TestRunsTarget(t *testing.T) {
	runs := runsTarget([]string{"A", "B", "C"}, nil, []string{"C", "B/SomeClass/testX"})
	require.True(t, runs("A"))
	require.True(t, runs("B"), "a skipped test must not exclude its target")
	require.False(t, runs("C"))
	require.False(t, runs("D"), "not in the xctestrun")

	runs = runsTarget(nil, []string{"B/SomeClass"}, nil)
	require.True(t, runs("B"))
	require.False(t, runs("A"))
}

func TestSummaryOutcomes(t *testing.T) {
	tc := func(result model3.TestResult, secs float64, retries ...model3.TestResult) model3.TestCaseWithRetries {
		c := model3.TestCaseWithRetries{TestCase: model3.TestCase{Result: result, Time: time.Duration(secs * float64(time.Second))}}
		for _, r := range retries {
			c.Retries = append(c.Retries, model3.TestCase{Result: r, Time: time.Second})
		}
		return c
	}
	summary := model3.TestSummary{TestPlans: []model3.TestPlan{{TestBundles: []model3.TestBundle{
		{Name: "CoreTests.xctest", TestSuites: []model3.TestSuite{{TestCases: []model3.TestCaseWithRetries{
			tc(model3.TestResultPassed, 0.5),
			tc(model3.TestResultFailed, 1, model3.TestResultFailed),
			tc(model3.TestResultPassed, 0.5, model3.TestResultFailed),
			tc(model3.TestResultSkipped, 9),
		}}}},
		{Name: "AppTests", TestSuites: []model3.TestSuite{{TestCases: []model3.TestCaseWithRetries{tc(model3.TestResultPassed, 2)}}}},
	}}}}

	require.Equal(t, []selector.UnitOutcome{
		{Key: "CoreTests", Tests: 3, Failed: 1, Flaky: 1, Duration: 4 * time.Second},
		{Key: "AppTests", Tests: 1, Duration: 2 * time.Second},
	}, summaryOutcomes(summary))
}

func TestXcresultOutcomesMissingBundle(t *testing.T) {
	_, err := xcresultOutcomes(filepath.Join(t.TempDir(), "missing.xcresult"))
	require.Error(t, err)
}
