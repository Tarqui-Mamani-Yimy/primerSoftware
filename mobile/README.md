# AI UML Architect mobile

This is the first mobile vertical slice: login, assigned projects, diagrams, versions-ready API access, and a focused UML workspace. The mobile app intentionally has no meetings or team-room features.

The Go runtime API defaults to `http://10.0.2.2:8080/api/v1`, which is the
Android emulator route to the host machine. Override it for a physical device
or another environment with `--dart-define=API_BASE_URL=...`.

The web client defaults to the same Go API on `http://localhost:8080/api/v1`.
Set `VITE_API_BASE_URL` in `frontend/.env` when the web client needs another
reachable Go runtime URL.

From this directory:

```bash
flutter pub get
flutter analyze
flutter test
flutter build apk --debug
```

For a physical Android device:

```bash
flutter run --dart-define=API_BASE_URL=http://<HOST_LAN_IP>:8080/api/v1
```

For iOS, run `flutter build ios --debug` on macOS with Xcode installed and
provide the reachable host URL through the same define. The mobile scope
intentionally excludes meetings and voice.
