void MainWindow::initUi() {
    QWidget *currentTab;
    QVBoxLayout *currentLayout;
    QWidget *fieldFrame;
    QHBoxLayout *fieldHorizontal;

    // ----------------------------------------------------------------
    // Tab Creation
    // ----------------------------------------------------------------
    QWidget *mainMenuTab = new QWidget();
    QVBoxLayout *mainMenuLayout = new QVBoxLayout(mainMenuTab);
    mainMenuLayout->setSpacing(10);
    mainMenuLayout->setContentsMargins(15, 15, 15, 15);

    // Use the new macro for the actual tab name
    QVBoxLayout *quickLaunchLayout = new QVBoxLayout();
    mainMenuLayout->addLayout(quickLaunchLayout);
    m_tabs->addTab(mainMenuTab, "TEMPLATE_MAIN_MENU_TAB_LABEL");


    // ----------------------------------------------------------------
    // Heading & Info Label Construction
    // ----------------------------------------------------------------
    QString headingText = "TEMPLATE_MAIN_MENU_HEADING_LABEL";
    QLabel *welcomeHeadingLabel = new QLabel();
    welcomeHeadingLabel->setObjectName("WelcomeHeadingLabel");

    if (headingText.isEmpty()) {
        welcomeHeadingLabel->setVisible(false); // Completely collapses the widget
    } else {
        welcomeHeadingLabel->setText(headingText);
        // Optional: Apply your specific stylesheet here
        welcomeHeadingLabel->setStyleSheet(appStrongLabelStyle("font-size: 16px;"));
    }
    mainMenuLayout->addWidget(welcomeHeadingLabel);

    QString infoText = "TEMPLATE_MAIN_MENU_INFO_LABEL";
    QLabel *infoDescriptionLabel = new QLabel();
    infoDescriptionLabel->setObjectName("InfoDescriptionLabel");

    if (infoText.isEmpty()) {
        infoDescriptionLabel->setVisible(false); // Completely collapses the widget
    } else {
        infoDescriptionLabel->setText(infoText);
        infoDescriptionLabel->setWordWrap(true); // Ensures long descriptions don't break the window width
        infoDescriptionLabel->setStyleSheet(appSubtleLabelStyle("font-size: 13px;"));
    }
    mainMenuLayout->addWidget(infoDescriptionLabel);

    // Push the dynamic labels to the top of the tab
    mainMenuLayout->addStretch();

    QPushButton *settingsBtn = new QPushButton("⚙️ Edit Runtime Defaults");
    settingsBtn->setObjectName("SettingsRuntimeDefaultsBtn");
    settingsBtn->setStyleSheet(appSecondaryButtonStyle());
    settingsBtn->setFocusPolicy(Qt::TabFocus);

    connect(settingsBtn, &QPushButton::clicked, this, &MainWindow::openSettingsEditor);
    mainMenuLayout->addWidget(settingsBtn);

    QPushButton *advancedSettingsBtn = new QPushButton("Advanced Settings");
    advancedSettingsBtn->setObjectName("AdvancedSettingsBtn");
    advancedSettingsBtn->setStyleSheet(appSimpleAccentButtonStyle(themeVal("#7f8c8d", "#5f6b76")));
    advancedSettingsBtn->setFocusPolicy(Qt::TabFocus);

    connect(advancedSettingsBtn, &QPushButton::clicked, this, &MainWindow::openAdvancedSettings);
    mainMenuLayout->addWidget(advancedSettingsBtn);

    QPushButton *exitAppBtn = new QPushButton("✖ Exit Application");
    exitAppBtn->setObjectName("ExitAppBtn");
    exitAppBtn->setStyleSheet(appSimpleAccentButtonStyle("#a34747"));
    exitAppBtn->setFocusPolicy(Qt::TabFocus);

    connect(exitAppBtn, &QPushButton::clicked, this, &MainWindow::cleanupAndExit);
    mainMenuLayout->addWidget(exitAppBtn);

    m_tabs->addTab(mainMenuTab, "TEMPLATE_MAIN_MENU_TAB_LABEL");

    if (qApp->arguments().contains("--settings")) {
        QTimer::singleShot(100, this, &MainWindow::openSettingsEditor);
    }
