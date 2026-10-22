package step

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult3"
	"github.com/bitrise-io/go-xcode/v2/testresult/xcresult3/model3"
	"github.com/bitrise-steplib/bitrise-step-test-selection/selector"
	"howett.net/plist"
)

// StartTestSelectionShadow works out which test targets a change-based test selection
// would have run. It never changes what the step runs, and it only acts when the
// backend injects a shadow configuration; otherwise it returns nil. The step has no
// project input, so the analysis uses the app's BITRISE_PROJECT_PATH and BITRISE_SCHEME.
func (s XcodebuildTester) StartTestSelectionShadow(config Config) *selector.Shadow {
	targets, err := xctestrunTargets(config.Xctestrun)
	if err != nil {
		s.logger.Debugf("Test selection shadow run: cannot list the xctestrun's test targets: %s", err)
	}
	runs := runsTarget(targets, config.OnlyTesting, config.SkipTesting)

	sh, ok := selector.StartShadow(selector.ShadowOptions{
		Step: "xcode-test-without-building",
		Config: selector.Config{
			ProjectType: "xcode",
			ProjectPath: os.Getenv("BITRISE_PROJECT_PATH"),
			Scheme:      os.Getenv("BITRISE_SCHEME"),
		},
		Keep: func(t selector.Test, _ string) bool { return runs(t.Name) },
		Logf: func(format string, v ...interface{}) {
			s.logger.Println()
			s.logger.Infof(format, v...)
		},
		Debugf: s.logger.Debugf,
	})
	if !ok {
		return nil
	}
	return sh
}

// FinishTestSelectionShadow reports each test target's results so the backend can tell
// what a selection would have saved and whether it would have skipped a failure.
func (s XcodebuildTester) FinishTestSelectionShadow(sh *selector.Shadow, xcresultPath string, testRun time.Duration) {
	if sh == nil {
		return
	}
	outcomes, err := xcresultOutcomes(xcresultPath)
	if err != nil {
		s.logger.Debugf("Test selection shadow run: cannot read the test results: %s", err)
	}
	sh.Finish(outcomes, testRun)
}

// xcresultOutcomes sums the xcresult's test cases per test target. A target's duration
// is its test cases' time including retries, so simulator boot is not counted and the
// saving is a lower bound.
func xcresultOutcomes(xcresultPath string) ([]selector.UnitOutcome, error) {
	if _, err := os.Stat(xcresultPath); err != nil {
		return nil, err
	}
	data, err := xcresult3.ParseTestResults(xcresultPath)
	if err != nil {
		return nil, err
	}
	summary, _, err := model3.Convert(data)
	if err != nil {
		return nil, err
	}
	return summaryOutcomes(*summary), nil
}

func summaryOutcomes(summary model3.TestSummary) []selector.UnitOutcome {
	var keys []string
	byTarget := map[string]*selector.UnitOutcome{}
	for _, plan := range summary.TestPlans {
		for _, bundle := range plan.TestBundles {
			key := strings.TrimSuffix(bundle.Name, ".xctest")
			o, ok := byTarget[key]
			if !ok {
				o = &selector.UnitOutcome{Key: key}
				byTarget[key] = o
				keys = append(keys, key)
			}
			for _, suite := range bundle.TestSuites {
				for _, tc := range suite.TestCases {
					addTestCase(o, tc)
				}
			}
		}
	}
	outcomes := make([]selector.UnitOutcome, 0, len(keys))
	for _, k := range keys {
		outcomes = append(outcomes, *byTarget[k])
	}
	return outcomes
}

func addTestCase(o *selector.UnitOutcome, tc model3.TestCaseWithRetries) {
	if tc.Result == model3.TestResultSkipped {
		return
	}
	o.Tests++
	o.Duration += tc.Time
	failedBefore := false
	for _, r := range tc.Retries {
		o.Duration += r.Time
		failedBefore = failedBefore || r.Result == model3.TestResultFailed
	}
	switch {
	case tc.Result == model3.TestResultFailed:
		o.Failed++
	case failedBefore:
		o.Flaky++
	}
}

// xctestrunTargets lists the test target names of an xctestrun file, format 1 or 2.
func xctestrunTargets(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]interface{}
	if _, err := plist.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var names []string
	if configs, ok := doc["TestConfigurations"].([]interface{}); ok {
		for _, c := range configs {
			cfg, _ := c.(map[string]interface{})
			targets, _ := cfg["TestTargets"].([]interface{})
			for _, t := range targets {
				target, _ := t.(map[string]interface{})
				if name, _ := target["BlueprintName"].(string); name != "" {
					names = append(names, name)
				}
			}
		}
		return names, nil
	}
	for key, value := range doc {
		if _, isDict := value.(map[string]interface{}); isDict && !strings.HasPrefix(key, "__") {
			names = append(names, key)
		}
	}
	return names, nil
}

// runsTarget reports whether the step runs any test of a target: it is in the
// xctestrun (when that could be read), named by only-testing (when given), and not
// skipped as a whole.
func runsTarget(xctestrun, onlyTesting, skipTesting []string) func(string) bool {
	set := func(ids []string, wholeTargetOnly bool) map[string]bool {
		out := map[string]bool{}
		for _, id := range ids {
			target, rest, _ := strings.Cut(strings.TrimSpace(id), "/")
			if wholeTargetOnly && rest != "" {
				continue
			}
			out[target] = true
		}
		return out
	}
	inXctestrun := set(xctestrun, false)
	only := set(onlyTesting, false)
	skipped := set(skipTesting, true)
	return func(name string) bool {
		if len(inXctestrun) > 0 && !inXctestrun[name] {
			return false
		}
		if len(only) > 0 && !only[name] {
			return false
		}
		return !skipped[name]
	}
}
