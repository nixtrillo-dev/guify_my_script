MainWindow::MainWindow(const QIcon &appIcon) : m_appIcon(appIcon), m_maxBtn(nullptr) {
    setWindowTitle("TEMPLATE_TITLE");
    resize(TEMPLATE_WIDTH, TEMPLATE_HEIGHT);
    setWindowIcon(m_appIcon);

    setWindowFlags(Qt::Window | Qt::FramelessWindowHint);
    qApp->installEventFilter(this);

    QWidget *centralWidget = new QWidget(this);
    centralWidget->setObjectName("centralWidget");
    centralWidget->setFocusPolicy(Qt::ClickFocus);
    m_mainLayout = new QVBoxLayout(centralWidget);
    m_mainLayout->setContentsMargins(1, 1, 1, 1);
    m_mainLayout->setSpacing(0);
    setCentralWidget(centralWidget);

    m_tabs = new QTabWidget(this);
    m_tabs->setFocusPolicy(Qt::TabFocus);

    if (QTabBar *tb = m_tabs->findChild<QTabBar*>()) {
        tb->setFocusPolicy(Qt::TabFocus);
    }

    m_trayMenu = new QMenu(this);
    QAction *toggleAct = new QAction("Show/Hide Panel", this);
    connect(toggleAct, &QAction::triggered, this, &MainWindow::toggleVisibility);
    m_trayMenu->addAction(toggleAct);

    m_trayMenu->addSeparator();
    QAction *settingsTrayAction = new QAction("⚙️ Edit Runtime Defaults", this);
    connect(settingsTrayAction, &QAction::triggered, this, &MainWindow::openSettingsEditor);
    m_trayMenu->addAction(settingsTrayAction);

    QAction *advancedTrayAction = new QAction("Advanced Settings", this);
    connect(advancedTrayAction, &QAction::triggered, this, &MainWindow::openAdvancedSettings);
    m_trayMenu->addAction(advancedTrayAction);
    m_trayMenu->addSeparator();

    setupCustomTitleBar();
    m_mainLayout->addWidget(m_tabs);

    // Core Initialization Chain
    initMetadata();      // Loads memory maps for dynamic refresh queries
    initRuntimeDefaults();
    applyThemeStyle();
    initUi();            // Creates the widgets and populates m_uiFields

    // Trigger dynamic field refreshes (Must run AFTER initUi populates widgets)
    for (auto it = m_fieldMetadata.begin(); it != m_fieldMetadata.end(); ++it) {
        QJsonObject endpoints = it.value()["json_endpoints"].toObject();
        if (!endpoints["get_defaults_endpoint"].toArray().isEmpty() ||
            !endpoints["get_enabled_endpoint"].toArray().isEmpty() ||
            !endpoints["array_endpoint"].toArray().isEmpty()) { // Also catch dynamic arrays

            QStringList parts = it.key().split("::");
            if (parts.size() == 2) {
                refreshField(parts[0], parts[1]);
            }
        }
    }

    initTray();

    QString serverName = "qt_app_" + APP_UUID;
    QLocalServer::removeServer(serverName);
    m_localServer = new QLocalServer(this);
    m_localServer->listen(serverName);

    connect(m_localServer, &QLocalServer::newConnection, this, [this]() {
        QLocalSocket *client = m_localServer->nextPendingConnection();
        connect(client, &QLocalSocket::readyRead, this, [this, client]() {
            QString actionLabel = QString::fromUtf8(client->readAll());
            if (m_quickActions.contains(actionLabel)) {
                m_quickActions[actionLabel]();
            }
            client->disconnectFromServer();
        });
        connect(client, &QLocalSocket::disconnected, client, &QLocalSocket::deleteLater);
    });

    centralWidget->setFocus();
}

MainWindow::~MainWindow() {
    cleanupAndExit();
}

void MainWindow::toggleVisibility() {
    if (m_settingsDialog) {
        m_settingsDialog->raise();
        m_settingsDialog->activateWindow();
        return;
    }

    if (isVisible() && !isActiveWindow()) {
        raise();
        activateWindow();
    } else if (isVisible()) {
        if (QSystemTrayIcon::isSystemTrayAvailable()) {
            hide();
            QTimer::singleShot(100, []() { malloc_trim(0); });
        } else {
            showMinimized();
        }
    } else {
        show();
        raise();
        activateWindow();
    }
}

Qt::Edges MainWindow::getResizeEdges(const QPoint &pos) {
    if (this->isMaximized()) return {};

    const int border = 8;
    int w = this->width();
    int h = this->height();

    Qt::Edges edges = {};
    if (pos.x() <= border) edges |= Qt::LeftEdge;
    if (pos.x() >= w - border) edges |= Qt::RightEdge;
    if (pos.y() <= border) edges |= Qt::TopEdge;
    if (pos.y() >= h - border) edges |= Qt::BottomEdge;

    return edges;
}

bool MainWindow::eventFilter(QObject *obj, QEvent *event) {
    if (obj->objectName() == "CustomTitleBar" && event->type() == QEvent::ContextMenu) {
        QContextMenuEvent *contextEvent = static_cast<QContextMenuEvent*>(event);
        if (QWidget *menuButton = qobject_cast<QWidget*>(obj->findChild<QWidget*>("WindowIconWidget"))) {
            if (menuButton->geometry().contains(contextEvent->pos())) {
                // The application-menu button handles right-click on the
                // press event; do not let its context event reach the
                // title-bar window-controls menu as well.
                event->accept();
                return true;
            }
        }
        showWindowControlMenu(this, contextEvent->globalPos(), m_maxBtn);
        event->accept();
        return true;
    }

    if (obj->objectName() == "WindowIconWidget" && event->type() == QEvent::ContextMenu) {
        // Its mouse-press handler already displayed the application menu.
        event->accept();
        return true;
    }

    if (obj->objectName() == "WindowIconWidget" && event->type() == QEvent::MouseButtonPress) {
        QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
        if (mouseEvent->button() == Qt::LeftButton || mouseEvent->button() == Qt::RightButton) {
            if (m_trayMenu) {
                m_trayMenu->popup(QCursor::pos());
            }
            event->accept();
            return true;
        }
    }

    if (event->type() == QEvent::MouseMove) {
        QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
        QPoint globalPos = mouseEvent->globalPosition().toPoint();
        QPoint localPos = this->mapFromGlobal(globalPos);

        if (this->rect().contains(localPos) || getResizeEdges(localPos)) {
            Qt::Edges edges = getResizeEdges(localPos);
            if (!edges) {
                if (cursor().shape() != Qt::ArrowCursor) {
                    setCursor(Qt::ArrowCursor);
                }
            } else {
                if ((edges & Qt::LeftEdge) && (edges & Qt::TopEdge)) setCursor(Qt::SizeBDiagCursor);
                else if ((edges & Qt::RightEdge) && (edges & Qt::BottomEdge)) setCursor(Qt::SizeBDiagCursor);
                else if ((edges & Qt::RightEdge) && (edges & Qt::TopEdge)) setCursor(Qt::SizeFDiagCursor);
                else if ((edges & Qt::LeftEdge) && (edges & Qt::BottomEdge)) setCursor(Qt::SizeFDiagCursor);
                else if (edges & Qt::LeftEdge || edges & Qt::RightEdge) setCursor(Qt::SizeHorCursor);
                else if (edges & Qt::TopEdge || edges & Qt::BottomEdge) setCursor(Qt::SizeVerCursor);
            }
        }
    }
    return QMainWindow::eventFilter(obj, event);
}

void MainWindow::mousePressEvent(QMouseEvent *event) {
    if (event->button() == Qt::LeftButton) {
        Qt::Edges edges = getResizeEdges(event->pos());
        if (edges) {
            if (this->windowHandle()) {
                this->windowHandle()->startSystemResize(edges);
            }
            event->accept();
            return;
        }
        else if (event->pos().y() <= 36) {
            if (this->windowHandle()) {
                this->windowHandle()->startSystemMove();
            }
            event->accept();
            return;
        }
    }
    QMainWindow::mousePressEvent(event);
}

void MainWindow::mouseDoubleClickEvent(QMouseEvent *event) {
    if (event->button() == Qt::LeftButton) {
        if (event->pos().y() <= 36 && event->pos().y() > 8) {
            if (this->isMaximized()) {
                this->showNormal();
                if (m_maxBtn) m_maxBtn->setText("🗖");
            } else {
                this->showMaximized();
                if (m_maxBtn) m_maxBtn->setText("🗗");
            }
            event->accept();
        }
    }
}

void MainWindow::closeEvent(QCloseEvent *event) {
    cleanupAndExit();
    event->accept();
}

void MainWindow::cleanupAndExit() {
    try { killpg(0, SIGTERM); } catch(...) {}
    if (active_lock_fd > 0) {
        flock(active_lock_fd, LOCK_UN);
        ::close(active_lock_fd);
        unlink(LOCK_FILE.c_str());
        active_lock_fd = -1;
    }
    QApplication::quit();
}
