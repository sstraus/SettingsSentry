# Release preparation: v1.2.1

## Contract

CI selects Go from `go.mod`, builds the lowercase `settingssentry` binary, and uploads files that exist. A release contains ZIP archives for both macOS architectures, the correct application version, and no `TestCommand.cfg` fixture.

The production state comes from the workflow files, Makefile, `.goreleaser.yml`, and `main.Version`. GitHub action input contracts were read from the upstream action metadata. Hosted runner execution and publication still need verification after approval.

## Test repair contract

`ProcessConfiguration` copies existing files even when an optional file is absent. It has no error return. The real filesystem supplies file contents; normal backup logs successful copies. The repaired test checks copied bytes, absence of a missing-file artifact, no error log, and successful-copy messages. Missing files are skipped silently by the early `Stat` check, including in dry-run. Production behavior does not change.

Cron tests must save the real `crontab -l` output before mutation and restore the original presence and contents in `t.Cleanup`, including after `t.Fatal`. CLI cron tests share this contract. Test packages must run serially because the crontab belongs to the account, not the package. These tests run only in GitHub CI, never on this Mac; their OS behavior remains unverified locally.

The pre-fix run at `9c97839` failed to compile `backup_test.go` (undefined `setup`, undefined `BackupTestDirs`, and use of a void result). Evidence is in the coordinator's `validation.log`. The previous assertions assumed an error API that production does not have, so the repair replaces that invalid expectation rather than changing production to satisfy it.

## Audit

The unpushed commit `83e00c9` makes CodeQL read the Go version from `go.mod`. It also downgrades setup-go and changes the workflow mode. It was preserved by rebasing it onto `origin/main` (`ed53397`) as `9c97839`. The subsequent CI update retains its Go-version fix and applies the audited setup-go update.

All ten open PRs were read with `gh pr view` and `gh pr diff`: #31, #33, #35, #36, #38, #41, #43, #45, #47, and #50. The existing major-tag style is retained; CodeQL retains its full version tag. Upstream metadata retains every input used here. The major updates use Node 24 on GitHub-hosted runners. Checkout v7 restricts unsafe fork checkout for events this repository does not use. Upload-artifact v7 retains archive mode by default. Codecov v7 changes its signature identity; its input names remain valid. The release action's old `distribution`, `version`, and `args` inputs were invalid even before the major update; these inputs belong to GoReleaser and are corrected separately.

### Observed failures

- CodeQL run `40159017615`: Go 1.24.13 cannot load the module, which requires Go 1.26.0, with `GOTOOLCHAIN=local`. Autobuild also invokes the Makefile help target instead of compiling.
- CI run `38444918096`: the unpinned curl installer fails its golangci-lint archive checksum check; tests fail on the same Go version mismatch. The macOS test job also invokes `apt-get`.
- Source audit: CI artifact paths use uppercase names, but Makefile emits lowercase names.
- Source audit: the release workflow passes unsupported inputs to softprops and does not upload its archives. Direct `go build` embeds any local command fixture. GoReleaser sets `main.version`, but the application variable is `main.Version`.

## Publication gate

Do not push, create a release tag, merge GitHub PRs, or publish before Boss says "vai". Keep the dirty main checkout unchanged. Run publication commands from the isolated release worktree, after the coordinator reviews the final commit and validation evidence.

### Regression check

`scripts/check-release.py` opens the real GoReleaser archives. It checks both Mach-O architectures and runs the native binary. It catches a wrong linker symbol by comparing the visible version with GoReleaser metadata. It catches fixture leaks in both ZIP contents and the embedded configuration filesystem through `configsinit`. Build the snapshot with a temporary `configs/TestCommand.cfg` present to exercise the exclusion path.

Adjacent scenarios checked in source: absent fixture, existing rename destination, failed build cleanup, and both archive architectures. The rename destination is never overwritten. A shell exit trap restores a fixture after the build or release command fails.

## Commands after approval

Run these commands only after Boss says "vai" and the coordinator accepts the commits. Do not run GoReleaser publication locally. The tag push invokes the Release workflow.

First push the reviewed branch to main, without a force push:

```sh
set -eu
cd /Users/stefano.straus/Gits/personal/SettingsSentry__wt/fix-actions-bumps
test "$(git branch --show-current)" = fix/actions-bumps
git diff HEAD --exit-code
git fetch origin
git merge-base --is-ancestor origin/main HEAD
release_commit=$(git rev-parse HEAD)
git push origin HEAD:main
```

Wait for the CI run for `$release_commit` on main. Confirm that the full unit and integration jobs pass, including the cron tests. Do not create the release tag if they fail. Select the matching run ID from:

```sh
gh run list --repo sstraus/SettingsSentry --workflow ci.yml --branch main --commit "$release_commit"
gh run watch <CI_RUN_ID> --repo sstraus/SettingsSentry --exit-status
```

Then publish the tag:

```sh
git tag -a v1.2.1 -m 'Release SettingsSentry v1.2.1'
git push origin refs/tags/v1.2.1
gh run list --repo sstraus/SettingsSentry --workflow release.yml --branch v1.2.1
gh run watch <RELEASE_RUN_ID> --repo sstraus/SettingsSentry --exit-status
gh release view v1.2.1 --repo sstraus/SettingsSentry
```

Confirm that both architecture ZIPs and `checksums.txt` are attached. Only then close the ten superseded action-bump PRs:

```sh
for pr in 31 33 35 36 38 41 43 45 47 50; do
  gh pr close "$pr" --repo sstraus/SettingsSentry --comment 'Integrated in the reviewed v1.2.1 CI and release update.'
done
```

### Homebrew notification limitation

The release workflow uses `GITHUB_TOKEN`. GitHub does not trigger another workflow from a release event created with that token. Thus the separate release-event workflow is not sufficient to notify the tap for this automated release. The coordinator must arrange an authorized tap dispatch after successful publication, or approve a separate workflow change. This preparation does not change that token or dispatch wiring. Source: [GitHub workflow trigger rules](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow#triggering-a-workflow-from-a-workflow), checked on 2026-10-26.

## Local validation (2026-10-26)

Evidence directory: `~/Gits/.tmp/settingssentry-release/`. Validation runs were sequential through the coordinator's build-slot wrapper on native macOS arm64, Go 1.27.3. No cron command was executed.

- `final-check.log`: `go test -p 1 -race -count=1 -skip '^(TestExecuteInstall|TestExecuteRemove)$'` on every package except `SettingsSentry/cron`: PASS (35 seconds). The same package selection with `-tags=integration`: PASS (34 seconds).
- `final-check.log`: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run ./...`: PASS, zero issues (5 seconds). The tool linker emits a macOS deployment-target warning; analysis succeeds. The previous network failure recovered on retry.
- `validation2.log`: actionlint v1.7.7 on all workflows: PASS (2 seconds); `goreleaser check`: PASS.
- `validation2.log`: `make zip VERSION=1.2.1` (includes `make build`): PASS. A deliberately stale ZIP containing `TestCommand.cfg` was replaced, and the resulting ZIP excludes it. An invalid real Go build flag caused the expected build failure and the shell trap restored the fixture.
- `validation2.log`: `make release` snapshot: PASS (2 seconds). `python3 scripts/check-release.py`: PASS (1 second). Both Mach-O architectures and both ZIP configuration sets were checked. The arm64 binary ran and extracted its embedded configs without the command fixture. The x86_64 binary was inspected, not executed.
- Snapshot metadata and the native binary both report `1.2.0-SNAPSHOT-9c97839`, as GoReleaser derives snapshots from the previous tag. These are test artifacts, not publishable v1.2.1 archives. The tag-triggered workflow derives the final version from `v1.2.1`.
- `git diff --check`: PASS. Go test files were formatted with `gofmt`. The `make zip VERSION=1.2.1` native binary also reports `SettingsSentry v1.2.1` with no arguments (`native-version.log`).

The only unverified test area is real cron integration: the cron package and CLI `TestExecuteInstall` / `TestExecuteRemove`. All were deliberately omitted locally and remain required in GitHub CI before tagging. The shared cleanup helper is type-checked by lint, but its OS restore behavior is not claimed as tested.

The instruction-link checker reports a pre-existing worktree violation: no canonical `AGENTS.md`, and tracked `CLAUDE.md` is a regular file. Both instruction sources were read. This release task does not convert them or change the dirty main checkout. No separate critic review has run in this child; the coordinator owns that review.
