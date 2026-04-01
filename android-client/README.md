# Android Client

Native Pipe Net client for Android, built with Kotlin and Jetpack Compose.

Current state:
- login by existing server username
- saved server address and login
- real mobile API integration with the Go backend
- chat list, message timeline, mentions inbox
- DM creation and user lookup
- reply, edit and delete message flows
- owner channel creation from mobile
- members sheet with invite/remove controls for admin and owner
- moderation log view
- periodic background refresh without clobbering an active draft

How it should feel:
- install an APK
- open the app
- enter server address and login once
- use it like a normal messenger

Backend requirement:
- run the Go server with mobile API enabled
- default mobile API endpoint is `http://<server>:8080`
- Android Emulator endpoint is `http://10.0.2.2:8080`

Local dev run:
1. Open `android-client/` in Android Studio.
2. Let Gradle sync.
3. Run the `app` configuration on an emulator or device.

CI build:
- GitHub Actions workflow `.github/workflows/android-debug-apk.yml` assembles a debug APK and uploads it as an artifact.
- This gives you a simple installable build path without requiring Android Studio for every user.
