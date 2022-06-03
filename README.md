# Xcode Test without building

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-xcode-test-without-building?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/releases)

Tests compiled bundles.

<details>
<summary>Description</summary>

Tests compiled bundles by running `xcodebuild test-without-building` command.
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `xctestrun` | Test run parameters file, generated during the build-for-testing action. | required |  |
| `destination` | Destination specifier describing the device (or devices) to use as a destination. | required |  |
| `xcodebuild_options` | Additional options to be added to the executed xcodebuild command. |  |  |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BITRISE_XCRESULT_PATH` | The result bundle path generated by `xcodebuild test-without-building`. |
| `BITRISE_XCRESULT_ZIP_PATH` | The zipped result bundle path generated by `xcodebuild test-without-building`. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-xcode-test-without-building/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
