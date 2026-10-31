void MainWindow::setupCustomTitleBar() {
    QWidget *titleBar = new QWidget(this);
    titleBar->setObjectName("CustomTitleBar");
    titleBar->setFixedHeight(36);
    titleBar->installEventFilter(this);

    QHBoxLayout *titleLayout = new QHBoxLayout(titleBar);
    titleLayout->setContentsMargins(12, 4, 6, 4);
    titleLayout->setSpacing(8);

    QWidget *iconContainer = new QWidget(titleBar);
    iconContainer->setObjectName("WindowIconWidget");
    iconContainer->setFixedSize(24, 24);
    iconContainer->setToolTip("Application menu");
    iconContainer->setCursor(Qt::PointingHandCursor);
    iconContainer->setAttribute(Qt::WA_Hover);
    QHBoxLayout *iconLayout = new QHBoxLayout(iconContainer);
    iconLayout->setContentsMargins(0, 0, 0, 0);

    QLabel *iconLabel = new QLabel(iconContainer);
    iconLabel->setObjectName("WindowIconLabel");
    iconLabel->setAlignment(Qt::AlignCenter);
    iconLabel->setAttribute(Qt::WA_TransparentForMouseEvents);
    const QPixmap iconPixmap = m_appIcon.pixmap(16, 16);
    if (iconPixmap.isNull()) {
        // Some desktops cannot resolve a themed application icon. Keep a
        // visible, familiar affordance for the application context menu.
        iconLabel->setText(QString::fromUtf8("▾"));
    } else {
        iconLabel->setPixmap(iconPixmap);
    }
    iconLayout->addWidget(iconLabel);
    titleLayout->addWidget(iconContainer);

    iconContainer->installEventFilter(this);

    QLabel *titleLabel = new QLabel(windowTitle(), this);
    titleLabel->setObjectName("WindowTitleLabel");
    titleLabel->setAttribute(Qt::WA_TransparentForMouseEvents);
    titleLabel->setStyleSheet(appSubtleLabelStyle("font-weight: bold; background: transparent;"));
    titleLayout->addWidget(titleLabel);

    titleLayout->addStretch();

    QPushButton *minBtn = new QPushButton("—", this);
    minBtn->setObjectName("TitleMinBtn");
    minBtn->setFixedSize(28, 24);
    connect(minBtn, &QPushButton::clicked, [this]() {
        if (TRAY_ON_MINIMIZE && QSystemTrayIcon::isSystemTrayAvailable()) {
            this->hide();
        } else {
            this->showMinimized();
        }
    });
    titleLayout->addWidget(minBtn);

    m_maxBtn = new QPushButton("🗖", this);
    m_maxBtn->setObjectName("TitleMaxBtn");
    m_maxBtn->setFixedSize(28, 24);
    connect(m_maxBtn, &QPushButton::clicked, [this]() {
        if (this->isMaximized()) {
            this->showNormal();
            m_maxBtn->setText("🗖");
        } else {
            this->showMaximized();
            m_maxBtn->setText("🗗");
        }
    });
    titleLayout->addWidget(m_maxBtn);

    QPushButton *closeBtn = new QPushButton("✕", this);
    closeBtn->setObjectName("TitleCloseBtn");
    closeBtn->setFixedSize(28, 24);
    connect(closeBtn, &QPushButton::clicked, [this]() {
        this->close();
    });
    titleLayout->addWidget(closeBtn);

    m_mainLayout->addWidget(titleBar);
}

QString MainWindow::escapeArg(const QString &arg) {
    QString s = arg;
    s.replace("\'", "\\'").replace("\"", "\\\"").replace("$", "\\$").replace("`", "\\`");
    return s;
}

void MainWindow::initTray() {
    if (!QSystemTrayIcon::isSystemTrayAvailable()) return;

    m_trayIcon = new QSystemTrayIcon(this);
    QIcon trayIcon = m_appIcon;
#ifdef Q_OS_WIN
    if (trayIcon.isNull()) {
        QPixmap fallbackPixmap(32, 32);
        fallbackPixmap.fill(Qt::transparent);
        QPainter painter(&fallbackPixmap);
        painter.setRenderHint(QPainter::Antialiasing);
        painter.setPen(Qt::NoPen);
        painter.setBrush(QColor("#287b9b"));
        painter.drawRoundedRect(QRectF(1, 1, 30, 30), 6, 6);
        painter.setPen(Qt::white);
        painter.setFont(QFont("Segoe UI", 18, QFont::Bold));
        painter.drawText(fallbackPixmap.rect(), Qt::AlignCenter, "G");
        trayIcon = QIcon(fallbackPixmap);
    }
#endif
    m_trayIcon->setIcon(trayIcon);
    m_trayIcon->setToolTip(windowTitle());

    m_trayMenu->addSeparator();
    QAction *quitAct = new QAction("Exit Application", this);
    connect(quitAct, &QAction::triggered, this, &MainWindow::cleanupAndExit);
    m_trayMenu->addAction(quitAct);

    m_trayIcon->setContextMenu(m_trayMenu);

    connect(m_trayIcon, &QSystemTrayIcon::activated, [this](QSystemTrayIcon::ActivationReason reason) {
        if (reason == QSystemTrayIcon::Trigger) {
            toggleVisibility();
        }
    });

    m_trayIcon->show();
}
