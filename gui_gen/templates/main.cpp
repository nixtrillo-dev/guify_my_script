int main(int argc, char *argv[]) {
    // Copyright (c) 2026 Nick Rowley
    // All rights reserved.
    // Use of this source code is governed by an MIT-style license that can be
    // found in the LICENSE file.
    if (argc >= 3 && std::string(argv[1]) == "--trigger") {
        QCoreApplication tmpApp(argc, argv);
        QLocalSocket socket;
        QString serverName = "qt_app_" + APP_UUID;
        socket.connectToServer(serverName);
        if (socket.waitForConnected(100)) {
            socket.write(argv[2]);
            socket.waitForBytesWritten(100);
        }
        return 0;
    }

#ifdef Q_OS_WIN
    // Let Qt select the native Windows platform theme. Forcing a generic or
    // inherited platform theme can crash qwindows.dll during window startup.
    qunsetenv("QT_QPA_PLATFORMTHEME");
#else
    qputenv("QT_QPA_PLATFORMTHEME", "generic");
#endif
    qputenv("QT_STYLE_OVERRIDE", "Fusion");

#ifndef Q_OS_WIN
    setpgid(0, 0);
#endif
    handle_instance_policy();

    QApplication app(argc, argv);

    // Set the internal application name using the sanitized UUID
    QCoreApplication::setApplicationName("TEMPLATE_PACKAGE_NAME");

    // Set the user-facing display name using your existing title macro
    QGuiApplication::setApplicationDisplayName("TEMPLATE_TITLE");

    // Set the desktop file name
    QString desktopFileName = "TEMPLATE_PACKAGE_NAME";
    QGuiApplication::setDesktopFileName(desktopFileName);

    app.setQuitOnLastWindowClosed(false);

    QIcon appIcon;
    QString iconPath = "TEMPLATE_ICON_RAW_PATH";
#ifdef Q_OS_WIN
    const QString windowsAppDir = QCoreApplication::applicationDirPath();
    for (const QString &suffix : {QStringLiteral("png"), QStringLiteral("ico"), QStringLiteral("svg"), QStringLiteral("jpg"), QStringLiteral("jpeg")}) {
        const QString bundledWindowsIcon = windowsAppDir + "/app_icon." + suffix;
        if (QFile::exists(bundledWindowsIcon)) {
            iconPath = bundledWindowsIcon;
            break;
        }
    }
#endif
    if (QFile::exists(iconPath)) {
        appIcon = QIcon(iconPath);
    } else {
        appIcon = QIcon::fromTheme(iconPath, QIcon::fromTheme("preferences-desktop"));
    }

    MainWindow win(appIcon);

    bool canStartHiddenInTray = QSystemTrayIcon::isSystemTrayAvailable();
#ifdef Q_OS_WIN
    canStartHiddenInTray = win.hasVisibleTrayIcon();
#endif
    if (!STARTUP_TO_TRAY || !canStartHiddenInTray) {
        win.show();
    } else {
        win.show();
        win.hide();
    }

    QTimer::singleShot(500, []() { releaseMemory(); });

    return app.exec();
}
