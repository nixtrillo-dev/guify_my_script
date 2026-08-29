int main(int argc, char *argv[]) {
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

    qputenv("QT_QPA_PLATFORMTHEME", "generic");
    qputenv("QT_STYLE_OVERRIDE", "Fusion");

    setpgid(0, 0);
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
    if (QFile::exists(iconPath)) {
        appIcon = QIcon(iconPath);
    } else {
        appIcon = QIcon::fromTheme(iconPath, QIcon::fromTheme("preferences-desktop"));
    }

    MainWindow win(appIcon);

    if (!STARTUP_TO_TRAY || !QSystemTrayIcon::isSystemTrayAvailable()) {
        win.show();
    } else {
        win.show();
        win.hide();
    }

    QTimer::singleShot(500, []() { malloc_trim(0); });

    return app.exec();
}
