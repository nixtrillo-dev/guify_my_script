static void showWindowControlMenu(QWidget *window, const QPoint &globalPos, QPushButton *maxButton) {
    QMenu menu(window);
    const bool maximized = window->isMaximized();

    QAction *minimizeAction = menu.addAction("Minimize");
    minimizeAction->setEnabled(!window->isMinimized());
    QAction *maximizeAction = menu.addAction(maximized ? "Restore" : "Maximize");
    menu.addSeparator();
    QAction *closeAction = menu.addAction("Close");

    QObject::connect(minimizeAction, &QAction::triggered, window, [window]() { window->showMinimized(); });
    QObject::connect(maximizeAction, &QAction::triggered, window, [window, maxButton, maximized]() {
        if (maximized) {
            window->showNormal();
            if (maxButton) maxButton->setText("🗖");
        } else {
            window->showMaximized();
            if (maxButton) maxButton->setText("🗗");
        }
    });
    QObject::connect(closeAction, &QAction::triggered, window, [window]() { window->close(); });
    menu.exec(globalPos);
}

#ifdef Q_OS_WIN
static QLockFile *active_instance_lock = nullptr;

struct InstanceWindowCloseRequest {
    DWORD processId;
};

static BOOL CALLBACK closeInstanceWindows(HWND window, LPARAM requestPtr) {
    auto *request = reinterpret_cast<InstanceWindowCloseRequest *>(requestPtr);
    DWORD windowProcessId = 0;
    GetWindowThreadProcessId(window, &windowProcessId);
    if (windowProcessId == request->processId) {
        PostMessageW(window, WM_CLOSE, 0, 0);
    }
    return TRUE;
}
#endif

void handle_instance_policy() {
    if (INSTANCE_MODE == "duplicate") return;

#ifdef Q_OS_WIN
    QString appKey = APP_UUID;
    appKey.replace(QRegularExpression("[^A-Za-z0-9_.-]"), "_");
    const QString lockPath = QDir::temp().filePath("guify_" + appKey + ".lock");
    active_instance_lock = new QLockFile(lockPath);
    if (active_instance_lock->tryLock(0)) return;

    if (INSTANCE_MODE == "single") {
        std::exit(0);
    }

    if (INSTANCE_MODE == "replace") {
        qint64 previousPid = 0;
        QString previousHost;
        QString previousApp;
        if (active_instance_lock->getLockInfo(&previousPid, &previousHost, &previousApp) &&
            previousPid > 0 && previousHost.compare(QSysInfo::machineHostName(), Qt::CaseInsensitive) == 0) {
            HANDLE previousProcess = OpenProcess(PROCESS_TERMINATE | SYNCHRONIZE, FALSE, static_cast<DWORD>(previousPid));
            if (previousProcess) {
                InstanceWindowCloseRequest request{static_cast<DWORD>(previousPid)};
                EnumWindows(closeInstanceWindows, reinterpret_cast<LPARAM>(&request));
                DWORD waitResult = WaitForSingleObject(previousProcess, 5000);
                if (waitResult == WAIT_TIMEOUT) {
                    TerminateProcess(previousProcess, 1);
                    waitResult = WaitForSingleObject(previousProcess, 5000);
                }
                CloseHandle(previousProcess);
                if (waitResult != WAIT_OBJECT_0) std::exit(1);
            }
        }

        if (!active_instance_lock->tryLock(10000)) {
            std::exit(1);
        }
    }
#else

    active_lock_fd = open(LOCK_FILE.c_str(), O_RDWR | O_CREAT, 0666);
    if (active_lock_fd < 0) return;

    if (flock(active_lock_fd, LOCK_EX | LOCK_NB) == 0) {
        ftruncate(active_lock_fd, 0);
        std::string pid_str = std::to_string(getpid()) + "\n";
        write(active_lock_fd, pid_str.c_str(), pid_str.length());
        return;
    }

    if (INSTANCE_MODE == "single") {
        if (active_lock_fd >= 0) ::close(active_lock_fd);
        exit(0);
    }

    if (INSTANCE_MODE == "replace") {
        std::ifstream infile(LOCK_FILE);
        pid_t old_pid = 0;
        if (infile >> old_pid && old_pid > 0) {
            kill(old_pid, SIGTERM);
            usleep(50000);
        }
        infile.close();
        if (active_lock_fd >= 0) ::close(active_lock_fd);

        active_lock_fd = open(LOCK_FILE.c_str(), O_RDWR | O_CREAT, 0666);
        if (flock(active_lock_fd, LOCK_EX) == 0) {
            ftruncate(active_lock_fd, 0);
            std::string pid_str = std::to_string(getpid()) + "\n";
            write(active_lock_fd, pid_str.c_str(), pid_str.length());
            return;
        }
        exit(1);
    }
#endif
}
