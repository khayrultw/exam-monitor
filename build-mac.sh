#!/bin/bash
set -e

APP_NAME="ExamGuardStudent"

echo "Building ${APP_NAME}..."
cd "$(dirname "$0")/client"
go build -o "${APP_NAME}" .

echo "Creating .app bundle..."
APP_DIR="../${APP_NAME}.app"
rm -rf "${APP_DIR}"
mkdir -p "${APP_DIR}/Contents/MacOS"
mkdir -p "${APP_DIR}/Contents/Resources"

mv "${APP_NAME}" "${APP_DIR}/Contents/MacOS/"

# Copy icon if available
if [ -f "appicon.png" ]; then
    cp appicon.png "${APP_DIR}/Contents/Resources/"
fi

cat > "${APP_DIR}/Contents/Info.plist" << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleDisplayName</key>
    <string>Exam Guard Student</string>
    <key>CFBundleExecutable</key>
    <string>ExamGuardStudent</string>
    <key>CFBundleIdentifier</key>
    <string>com.forward.ExamMonitorStudent</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>ExamGuardStudent</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.education</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.3</string>
    <key>NSScreenCaptureUsageDescription</key>
    <string>Exam Guard needs screen recording access to share your screen with the exam monitor.</string>
</dict>
</plist>
PLIST

echo "PkgInfo..."
echo -n "APPL????" > "${APP_DIR}/Contents/PkgInfo"

echo ""
echo "Done! Created ${APP_DIR}"
echo ""
echo "To run:"
echo "  open ${APP_DIR}"
echo ""
echo "Note: Grant Screen Recording permission when prompted."
