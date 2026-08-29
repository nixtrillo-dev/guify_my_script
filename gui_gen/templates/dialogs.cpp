void MainWindow::openGlobalHotkeysMenu() {
    QList<QPair<QString, QStringList>> candidates = {
        {"systemsettings", QStringList() << "kcm_keys"},
        {"systemsettings5", QStringList() << "kcm_keys"},
        {"kcmshell6", QStringList() << "kcm_keys"},
        {"kcmshell5", QStringList() << "kcm_keys"},
        {"gnome-control-center", QStringList() << "keyboard"},
        {"xfce4-keyboard-settings", QStringList()},
        {"cinnamon-settings", QStringList() << "keyboard"},
        {"mate-keybinding-properties", QStringList()},
        {"lxqt-config-globalkeyshortcuts", QStringList()}
    };

    for (const auto &candidate : candidates) {
        if (!QStandardPaths::findExecutable(candidate.first).isEmpty()) {
            QProcess::startDetached(candidate.first, candidate.second);
            return;
        }
    }

    QMessageBox::information(this, "Global Hotkeys",
        "No common desktop hotkeys settings panel was found. Copy a --trigger command below and assign it with your desktop environment's keyboard shortcuts tool.");
}

void MainWindow::openAdvancedSettings() {
    if (m_advancedSettingsDialog) {
        m_advancedSettingsDialog->raise();
        m_advancedSettingsDialog->activateWindow();
        return;
    }

    m_advancedSettingsDialog = new AppDialog(this);
    m_advancedSettingsDialog->setWindowTitle("Advanced Settings");
    m_advancedSettingsDialog->resize(750, 550);

    QVBoxLayout *advancedSettingsLayout = m_advancedSettingsDialog->contentLayout();
    advancedSettingsLayout->setSpacing(12);
    advancedSettingsLayout->setContentsMargins(20, 20, 20, 20);

    QLabel *advancedTitle = new QLabel("<h3 style='margin-bottom: 0px;'>Advanced Settings</h3>");
    advancedTitle->setObjectName("AdvancedTitleLabel");
    advancedTitle->setStyleSheet(appStrongLabelStyle());
    advancedTitle->setAlignment(Qt::AlignCenter);
    advancedSettingsLayout->addWidget(advancedTitle);

    // --- Single Unified Tab Widget for EVERYTHING ---
    QTabWidget *advancedTabs = new QTabWidget(m_advancedSettingsDialog);
    advancedTabs->setObjectName("advancedToolsTabWidget"); // Re-using standard identifier
    advancedSettingsLayout->addWidget(advancedTabs, 1);

    // --- Tab 1: Persistent 'General' Tab ---
    QWidget *generalTab = new QWidget();
    QVBoxLayout *generalTabLayout = new QVBoxLayout(generalTab);
    generalTabLayout->setSpacing(12);
    generalTabLayout->setContentsMargins(10, 10, 10, 10);

    QPushButton *openHotkeysBtn = new QPushButton("Open Global Hotkeys Menu");
    openHotkeysBtn->setObjectName("OpenHotkeysBtn");
    openHotkeysBtn->setStyleSheet(appSimpleAccentButtonStyle("#10a37f"));
    openHotkeysBtn->setFocusPolicy(Qt::TabFocus);
    connect(openHotkeysBtn, &QPushButton::clicked, this, &MainWindow::openGlobalHotkeysMenu);
    generalTabLayout->addWidget(openHotkeysBtn);

    QScrollArea *triggerScrollArea = new QScrollArea();
    triggerScrollArea->setWidgetResizable(true);
    triggerScrollArea->setFrameShape(QFrame::NoFrame);

    QWidget *triggerListWidget = new QWidget();
    QVBoxLayout *triggerListLayout = new QVBoxLayout(triggerListWidget);
    triggerListLayout->setSpacing(10);
    triggerListLayout->setContentsMargins(0, 0, 0, 0);

TEMPLATE_ADVANCED_TRIGGER_ROWS

    triggerListLayout->addStretch();
    triggerScrollArea->setWidget(triggerListWidget);
    generalTabLayout->addWidget(triggerScrollArea, 1);

    // Add 'General' as the first tab in the unified widget
    advancedTabs->addTab(generalTab, "General");

    // --- Subsequent Tabs (e.g., Global Hotkeys, etc.) injected via Go ---
TEMPLATE_ADVANCED_TOOLS_TABS

    // --- Bottom Action Bar ---
    QHBoxLayout *buttonLayout = new QHBoxLayout();
    QPushButton *closeBtn = new QPushButton("Close");
    closeBtn->setFocusPolicy(Qt::TabFocus);
    buttonLayout->addStretch();
    buttonLayout->addWidget(closeBtn);
    advancedSettingsLayout->addLayout(buttonLayout);

    connect(closeBtn, &QPushButton::clicked, m_advancedSettingsDialog, &QDialog::accept);

    m_advancedSettingsDialog->exec();
    m_advancedSettingsDialog->deleteLater();
    m_advancedSettingsDialog = nullptr;
}
