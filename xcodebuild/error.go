package xcodebuild

type XcodebuildError struct {
	Reason string
	Err    error
	Log    string
}

func (err XcodebuildError) Error() string {
	return err.Reason
}
