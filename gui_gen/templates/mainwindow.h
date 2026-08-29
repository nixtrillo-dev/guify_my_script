class MainWindow : public QMainWindow {
public:
    MainWindow(const QIcon &appIcon);
    ~MainWindow();

    void toggleVisibility();
    Qt::Edges getResizeEdges(const QPoint &pos);

protected:
    bool eventFilter(QObject *obj, QEvent *event) override;
    void mousePressEvent(QMouseEvent *event) override;
    void mouseDoubleClickEvent(QMouseEvent *event) override;
    void closeEvent(QCloseEvent *event) override;

private:
    QIcon m_appIcon;
    QVBoxLayout *m_mainLayout;
    QTabWidget *m_tabs;
    QSystemTrayIcon *m_trayIcon;
    QMenu *m_trayMenu;
    QPushButton *m_maxBtn;

    QMap<QString, QWidget*> m_uiFields;
    QMap<QString, QJsonObject> m_fieldMetadata;
    QMap<QString, QString> m_actionScripts;
    AppDialog *m_settingsDialog = nullptr;
    AppDialog *m_advancedSettingsDialog = nullptr;
    void openSettingsEditor();
    void openAdvancedSettings();
    void openGlobalHotkeysMenu();

    QMap<QString, QString> m_runtimeDefaults;
    QLocalServer *m_localServer = nullptr;
    QMap<QString, std::function<void()>> m_quickActions;

    // Metadata and runtime defaults
    void initMetadata();
    void initRuntimeDefaults();
    QString getRuntimeDefault(const QString& actionLabel, const QString& fieldName, const QString& fallback);

    // JSON endpoint helpers
    QJsonDocument fetchJsonEndpoint(const QStringList& endpointArgs, const QString& targetScript = SOURCE_SCRIPT);
    QJsonDocument fetchJsonEndpoint(const QJsonArray& endpointArr, const QString& actionLabel);
    QString resolveDynamicPath(QString rawPath, const QString& actionLabel);

    // Widget value helpers
    void applyWidgetValue(QWidget* targetWidget, const QString& newVal);
    void clearWidgetValue(QWidget* targetWidget);
    void updateWidgetOptions(QWidget* targetWidget, const QJsonObject& meta, const QJsonObject& endpoints, const QString& actionLabel);
    void updateWidgetDefaultValue(QWidget* targetWidget, const QJsonObject& meta, const QJsonObject& endpoints, const QString& actionLabel, const QString& fieldName);
    void updateWidgetEnabledState(QWidget* targetWidget, const QJsonObject& endpoints, const QString& actionLabel);
    void refreshField(const QString& actionLabel, const QString& fieldName);

    // UI initialization
    void initUi();
    void setupCustomTitleBar();
    void initTray();
    QString escapeArg(const QString &arg);

    // Theme handling
    QString currentThemeColor();
    QString readThemeCommand(const QString &program, const QStringList &arguments, int timeoutMs = 700) const;
    QString detectSystemThemeColor() const;
    void applyThemeStyle();
    void refreshThemedOverrides();

    // Cleanup
    void cleanupAndExit();
};
