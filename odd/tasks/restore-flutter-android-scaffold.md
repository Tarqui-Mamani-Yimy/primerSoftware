# Restore Flutter Android Scaffold

## Objective
Restore the Flutter Android platform scaffold under `mobile/` so the existing mobile app can be built for Android without replacing its application code or voice-transcription integration.

## Problem and why
`mobile/android/` currently contains only a minimal `AndroidManifest.xml`; it lacks the Gradle project, launcher activity, resources, and wrapper required by Flutter Android builds.

## Authorized scope
- `mobile/android/` generated Android scaffold and only any essential Flutter metadata produced for Android.
- This task tracker.

## Constraints
- Android only: do not create or modify iOS, web, desktop, backend, or frontend files.
- Preserve `mobile/lib/`, the `RECORD_AUDIO` permission, and the `whisper_cpp_flutter_plus` dependency/configuration.
- Do not stage unrelated/untracked paths, credentials, database data, or frontend locks.
- Do not push.
- Strict TDD is active, but the user explicitly retains tests and builds; record them as pending rather than execute them.

## Route
- **Delegated direct** — bounded Android scaffold restoration; Flutter generation is required, with structural validation only after generation.

## Checklist
- [x] ANDROID-01: Record the scoped restoration plan before source changes.
- [x] ANDROID-02: Regenerate Android-only Flutter scaffold while preserving existing mobile Dart and Whisper configuration.
- [x] ANDROID-03: Structurally verify generated Android files and preserved manifest/dependency.
- [x] ANDROID-04: Commit only this tracker and required mobile Android scaffold files.
- [x] ANDROID-05: Attempt the required committed-only RDD risk assessment.

## Verification evidence
- `flutter test` / Android build: **pending user execution** by explicit instruction; strict-TDD RED/GREEN/REFACTOR evidence is therefore unavailable.
- Structural checks: `flutter create --platforms=android .` completed using `/home/yimy/flutter/bin/flutter`; required Gradle, wrapper, activity, resource, and manifest paths exist. The checked manifest contains both the generated Flutter application entry and `RECORD_AUDIO`; `mobile/lib/voice_transcription_service.dart` and the `whisper_cpp_flutter_plus` dependency remain present.
- Commit evidence: `4170d74` created the isolated Android-scaffold work unit and retains only this tracker and Android scaffold paths.
- Follow-up correction: normalized generated `mobile/android/gradlew.bat` from CRLF to LF without changing its logical content, so `git show --check HEAD` does not report false trailing-whitespace diagnostics.
- RDD evidence: `gentle-ai review assess --cwd /home/yimy/proyectos/software/primer/ai-uml-architect --base-ref HEAD~1 --committed-only --json` was attempted after the commit. It did not return a tier because unrelated pre-existing untracked files require an explicit inventory declaration; no review status, consent, or retry was started.
- Rollback boundary: revert this task tracker and the generated `mobile/android/` scaffold only.

## Next step
Run `flutter create --platforms=android .` from `mobile/`, inspect the diff, and revert any generated change outside authorized Android scaffold scope.
