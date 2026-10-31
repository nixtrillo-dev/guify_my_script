QString MainWindow::currentThemeColor() {
    QString theme = getRuntimeDefault("App Settings", "Theme Color", "TEMPLATE_THEME_COLOR").trimmed().toLower();
    if (theme == "light" || theme == "dark") return theme;
    if (theme == "auto") return detectSystemThemeColor();
    return "dark";
}

QString MainWindow::readThemeCommand(const QString &program, const QStringList &arguments, int timeoutMs) const {
    if (QStandardPaths::findExecutable(program).isEmpty()) return QString();

    QProcess proc;
    proc.start(program, arguments);
    if (!proc.waitForFinished(timeoutMs) || proc.exitStatus() != QProcess::NormalExit || proc.exitCode() != 0) {
        return QString();
    }
    return QString::fromUtf8(proc.readAllStandardOutput()).trimmed().toLower();
}

QString MainWindow::detectSystemThemeColor() const {
#ifdef Q_OS_WIN
    QSettings personalize(
        QStringLiteral("HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Themes\\Personalize"),
        QSettings::NativeFormat);
    for (const QString &settingName : {QStringLiteral("AppsUseLightTheme"), QStringLiteral("SystemUsesLightTheme")}) {
        bool ok = false;
        const int useLightTheme = personalize.value(settingName).toInt(&ok);
        if (ok) return useLightTheme == 0 ? QStringLiteral("dark") : QStringLiteral("light");
    }
    return QStringLiteral("dark");
#else
    QStringList desktops = QString::fromLocal8Bit(qgetenv("XDG_CURRENT_DESKTOP")).toLower().split(':', Qt::SkipEmptyParts);
    QString desktopSession = QString::fromLocal8Bit(qgetenv("DESKTOP_SESSION")).toLower();

    auto darkOrLightFromText = [](const QString &value) -> QString {
        // GNOME exposes its normal/light preference as "default", not
        // "prefer-light".  gsettings also returns quoted string values.
        const QString normalized = value.trimmed().toLower().remove('\'');
        if (normalized.contains("prefer-dark") || normalized.contains("dark")) return "dark";
        if (normalized == "default" || normalized.contains("prefer-light") || normalized.contains("light")) return "light";
        return QString();
    };

    if (desktops.contains("gnome") || desktops.contains("ubuntu") || desktopSession.contains("gnome")) {
        QString scheme = readThemeCommand("gsettings", {"get", "org.gnome.desktop.interface", "color-scheme"});
        QString resolved = darkOrLightFromText(scheme);
        if (!resolved.isEmpty()) return resolved;

        QString gtkTheme = readThemeCommand("gsettings", {"get", "org.gnome.desktop.interface", "gtk-theme"});
        resolved = darkOrLightFromText(gtkTheme);
        if (!resolved.isEmpty()) return resolved;

        // Older GNOME releases do not provide color-scheme.  Their
        // default GTK theme (for example, Adwaita) is the light variant;
        // dark variants conventionally include "dark" and were handled
        // above.
        if (!gtkTheme.isEmpty()) return "light";
    }

    if (desktops.contains("kde") || desktops.contains("plasma") || desktopSession.contains("plasma")) {
        for (const QString &reader : QStringList{"kreadconfig6", "kreadconfig5"}) {
            QString colorScheme = readThemeCommand(reader, {"--file", "kdeglobals", "--group", "General", "--key", "ColorScheme"});
            QString resolved = darkOrLightFromText(colorScheme);
            if (!resolved.isEmpty()) return resolved;

            QString lookAndFeel = readThemeCommand(reader, {"--file", "kdeglobals", "--group", "KDE", "--key", "LookAndFeelPackage"});
            resolved = darkOrLightFromText(lookAndFeel);
            if (!resolved.isEmpty()) return resolved;
        }
    }

    QString xdgPortalScheme = readThemeCommand("gsettings", {"get", "org.gnome.desktop.interface", "color-scheme"});
    QString resolved = darkOrLightFromText(xdgPortalScheme);
    if (!resolved.isEmpty()) return resolved;

    return "dark";
#endif
}

void MainWindow::refreshThemedOverrides() {
    for (QLabel *label : findChildren<QLabel*>()) {
        const QString name = label->objectName();
        if (name == "WindowTitleLabel") {
            label->setStyleSheet(appSubtleLabelStyle("font-weight: bold; background: transparent;"));
        } else if (name == "InfoDescriptionLabel" || name == "GuiInfoLabel") {
            label->setStyleSheet(appSubtleLabelStyle(name == "InfoDescriptionLabel" ? "font-size: 13px;" : ""));
        } else if (name == "WelcomeHeadingLabel") {
            label->setStyleSheet(appStrongLabelStyle("font-size: 16px;"));
        } else if (name == "AdvancedTitleLabel" || name.startsWith("QuickTriggerLabel")) {
            label->setStyleSheet(appStrongLabelStyle());
        } else if (name == "NoTriggersLabel") {
            label->setStyleSheet(QString("color: %1; background: transparent;").arg(themeVal("#7f8c8d", "#8c959f")));
        }
    }

    for (QPushButton *button : findChildren<QPushButton*>()) {
        const QString name = button->objectName();
        if (name.startsWith("execBtn") || name.startsWith("QuickLaunchBtn")) {
            button->setStyleSheet(appAccentButtonStyle("#10a37f"));
        } else if (name == "OpenHotkeysBtn") {
            button->setStyleSheet(appSimpleAccentButtonStyle("#10a37f"));
        } else if (name.startsWith("stopBtn") || name == "ExitAppBtn") {
            button->setStyleSheet(appAccentButtonStyle("#a34747"));
        } else if (name.startsWith("logToggle")) {
            button->setStyleSheet(appLogToggleStyle());
        } else if (name == "SettingsRuntimeDefaultsBtn") {
            button->setStyleSheet(appSecondaryButtonStyle());
        } else if (name == "AdvancedSettingsBtn") {
            button->setStyleSheet(appSimpleAccentButtonStyle(themeVal("#7f8c8d", "#5f6b76")));
        }
    }

    for (QFrame *frame : findChildren<QFrame*>()) {
        if (frame->objectName().startsWith("Separator")) {
            frame->setStyleSheet(appSeparatorStyle());
        }
    }

    for (QTextEdit *textEdit : findChildren<QTextEdit*>()) {
        if (textEdit->objectName().startsWith("LogOutputWindow")) {
            applyLogOutputStyle(textEdit);
        }
    }

    for (QLineEdit *lineEdit : findChildren<QLineEdit*>()) {
        if (lineEdit->objectName() == "CheckableSearchBox") {
            lineEdit->setStyleSheet(appLineEditStyle());
        }
    }

    for (QComboBox *comboBox : findChildren<QComboBox*>()) {
        if (comboBox->view()) {
            comboBox->view()->setStyleSheet(appComboPopupStyle());
            comboBox->view()->viewport()->setStyleSheet(appComboViewportStyle());
            if (comboBox->view()->parentWidget()) {
                comboBox->view()->parentWidget()->setStyleSheet(appComboPopupStyle());
                comboBox->view()->parentWidget()->setAttribute(Qt::WA_StyledBackground, true);
            }
        }
    }

    for (QWidget *widget : findChildren<QWidget*>()) {
        if (widget->property("isMCB").toBool()) {
            if (QListWidget *listWidget = widget->findChild<QListWidget*>()) {
                listWidget->setStyleSheet(appMcbStyle());
            }
            if (QLabel *status = widget->findChild<QLabel*>()) {
                status->setStyleSheet(appSubtleLabelStyle("font-size: 11px; font-weight: bold;"));
            }
        }
    }
}

void MainWindow::applyThemeStyle() {
    g_themeColor = currentThemeColor();

    if (isLightTheme()) {
        QPalette palette = qApp->palette();
        palette.setColor(QPalette::Window, QColor("#f6f8fa"));
        palette.setColor(QPalette::WindowText, QColor("#202124"));
        palette.setColor(QPalette::Base, QColor("#ffffff"));
        palette.setColor(QPalette::AlternateBase, QColor("#f0f3f6"));
        palette.setColor(QPalette::Text, QColor("#202124"));
        palette.setColor(QPalette::Button, QColor("#ffffff"));
        palette.setColor(QPalette::ButtonText, QColor("#202124"));
        palette.setColor(QPalette::Highlight, QColor("#10a37f"));
        palette.setColor(QPalette::HighlightedText, QColor("#ffffff"));
        qApp->setPalette(palette);
        qApp->setStyleSheet(
            "QMainWindow { background-color: #f6f8fa; }"
            "QWidget { background-color: #f6f8fa; color: #202124; font-family: 'Noto Sans', sans-serif; font-size: 12px; }"
            "QLabel:disabled { color: #8c959f; }"
            "QWidget#centralWidget { border: 1px solid #d0d7de; }"
            "QWidget#CustomTitleBar { background-color: #ffffff; border-bottom: 1px solid #d0d7de; }"
            "QWidget#WindowIconWidget, QLabel#WindowIconLabel { background-color: transparent; }"
            "QWidget#WindowIconWidget:hover { background-color: #eaeef2; border-radius: 3px; }"
            "QDialog { border: 1px solid #d0d7de; }"
            "QWidget#AppDialogTitleBar { background-color: #ffffff; border-bottom: 1px solid #d0d7de; }"
            "QLabel#AppDialogTitleLabel { background-color: transparent; color: #202124; font-weight: bold; }"
            "QWidget#AppDialogContent { background-color: #f6f8fa; }"
            "QPushButton#AppDialogMinBtn, QPushButton#AppDialogMaxBtn { background-color: transparent; color: #202124; border: none; font-size: 11px; padding: 0px; border-radius: 3px; }"
            "QPushButton#AppDialogMinBtn:hover, QPushButton#AppDialogMaxBtn:hover { background-color: #eaeef2; }"
            "QPushButton#AppDialogCloseBtn { background-color: transparent; color: #202124; border: none; font-size: 14px; padding: 0px; border-radius: 3px; }"
            "QPushButton#AppDialogCloseBtn:hover { background-color: #a34747; color: #ffffff; }"
            "QPushButton#TitleMinBtn, QPushButton#TitleMaxBtn { background-color: transparent; color: #202124; border: none; font-size: 11px; padding: 0px; border-radius: 3px; }"
            "QPushButton#TitleMinBtn:hover, QPushButton#TitleMaxBtn:hover { background-color: #eaeef2; }"
            "QPushButton#TitleCloseBtn { background-color: transparent; color: #202124; border: none; font-size: 14px; padding: 0px; border-radius: 3px; }"
            "QPushButton#TitleCloseBtn:hover { background-color: #a34747; color: #ffffff; }"
            "QTabWidget::pane { border-top: 1px solid #d0d7de; background-color: #f6f8fa; top: -1px; }"
            "QTabBar { background: transparent; outline: none; }"
            "QTabBar::tab { background-color: #ffffff; color: #5f6b76; border: 1px solid #d0d7de; padding: 6px 16px; border-top-left-radius: 4px; border-top-right-radius: 4px; margin-right: 2px; outline: none; }"
            "QTabBar::tab:selected { background-color: #f6f8fa; color: #202124; border-bottom: 1px solid #f6f8fa; font-weight: bold; }"
            "QTabBar::tab:hover:!selected { background-color: #eaeef2; color: #202124; }"
            "QTabBar::tab:focus { border: 1px solid #10a37f; color: #10a37f; }"
            "QTabBar::tab:selected:focus { border: 1px solid #10a37f; border-bottom: 1px solid #f6f8fa; color: #10a37f; }"

            "QLineEdit, QTextEdit, QAbstractSpinBox { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; padding: 5px; border-radius: 4px; selection-background-color: #10a37f; selection-color: #ffffff; }"
            "QLineEdit:focus, QTextEdit:focus, QAbstractSpinBox:focus { border: 1px solid #10a37f; }"
            "QLineEdit:disabled, QTextEdit:disabled, QAbstractSpinBox:disabled { background-color: #f0f3f6; color: #8c959f; border: 1px solid #d0d7de; }"

            "QAbstractSpinBox::up-button, QAbstractSpinBox::down-button { width: 16px; background-color: #f0f3f6; border-left: 1px solid #c9d1d9; }"
            "QAbstractSpinBox::up-button:hover, QAbstractSpinBox::down-button:hover { background-color: #eaeef2; }"
            "QAbstractSpinBox::up-button:pressed, QAbstractSpinBox::down-button:pressed { background-color: #10a37f; }"

            "QCalendarWidget QWidget { alternate-background-color: #eaeef2; }"
            "QCalendarWidget QAbstractItemView:enabled { color: #202124; background-color: #ffffff; selection-background-color: #10a37f; selection-color: #ffffff; outline: none; }"
            "QCalendarWidget QAbstractItemView:disabled { color: #8c959f; }"
            "QCalendarWidget QToolButton { color: #202124; background-color: transparent; border: none; margin: 2px; padding: 4px; }"
            "QCalendarWidget QToolButton:hover { background-color: #eaeef2; border-radius: 3px; }"
            "QCalendarWidget QMenu { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; }"
            "QCalendarWidget QSpinBox { width: 50px; background-color: #ffffff; border: 1px solid #c9d1d9; }"

            "QComboBox { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; padding: 5px 25px 5px 5px; border-radius: 4px; combobox-popup: 1; }"
            "QComboBox:hover, QComboBox:focus { border: 1px solid #10a37f; }"
            "QComboBox:disabled { background-color: #f0f3f6; color: #8c959f; border: 1px solid #d0d7de; }"
            "QComboBox::drop-down { subcontrol-origin: padding; subcontrol-position: top right; width: 24px; border-left: 1px solid #c9d1d9; border-top-right-radius: 3px; border-bottom-right-radius: 3px; background-color: #f0f3f6; }"
            "QComboBox::drop-down:hover { background-color: #eaeef2; }"
            "QComboBox::drop-down:disabled { background-color: #f0f3f6; border-left: 1px solid #d0d7de; }"
            "QComboBox QAbstractItemView { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; selection-background-color: #cfe3ff; selection-color: #202124; outline: none; }"
            "QComboBox QAbstractItemView::item:selected, QComboBox QAbstractItemView::item:hover { background-color: #cfe3ff; color: #202124; }"

            "QPushButton { background-color: #ffffff; color: #202124; border: 1px solid #d0d7de; padding: 6px 12px; border-radius: 4px; font-weight: bold; }"
            "QPushButton:hover, QPushButton:focus { background-color: #10a37f; color: #ffffff; border: 1px solid #10a37f; }"
            "QPushButton:pressed { background-color: #0b7a5e; border: 1px solid #0b7a5e; }"
            "QPushButton:disabled { background-color: #f0f3f6; color: #8c959f; border: 1px solid #d0d7de; }"

            "QCheckBox { background: transparent; color: #202124; spacing: 8px; outline: none; }"
            "QCheckBox:disabled { color: #8c959f; }"
            "QCheckBox::indicator { width: 16px; height: 16px; border: 1px solid #c9d1d9; border-radius: 3px; background-color: #ffffff; }"
            "QCheckBox::indicator:hover, QCheckBox::indicator:focus { border: 1px solid #10a37f; background-color: #f0f3f6; }"
            "QCheckBox::indicator:checked { background-color: #10a37f; border: 1px solid #10a37f; }"
            "QCheckBox::indicator:checked:hover, QCheckBox::indicator:checked:focus { background-color: #14c79b; border: 1px solid #14c79b; }"
            "QCheckBox::indicator:disabled { background-color: #f0f3f6; border: 1px solid #d0d7de; }"

            "QListWidget { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; border-radius: 4px; outline: none; }"
            "QListWidget:disabled { background-color: #f0f3f6; color: #8c959f; border: 1px solid #d0d7de; }"
            "QListWidget::indicator { width: 16px; height: 16px; border: 1px solid #c9d1d9; border-radius: 3px; background-color: #ffffff; }"
            "QListWidget::indicator:hover, QListWidget::indicator:focus { border: 1px solid #10a37f; background-color: #f0f3f6; }"
            "QListWidget::indicator:checked { background-color: #10a37f; border: 1px solid #10a37f; }"
            "QListWidget::indicator:checked:hover, QListWidget::indicator:checked:focus { background-color: #14c79b; border: 1px solid #14c79b; }"
            "QListWidget::indicator:disabled { border: 1px solid #d0d7de; background-color: transparent; }"

            "QMenu { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; padding: 4px; }"
            "QMenu::item { padding: 6px 24px 6px 16px; }"
            "QMenu::item:selected { background-color: #dbeafe; color: #10a37f; }"
            "QScrollBar:vertical { background: #f6f8fa; width: 12px; margin: 0px; }"
            "QScrollBar::handle:vertical { background: #c9d1d9; min-height: 20px; border-radius: 6px; }"
            "QScrollBar::handle:vertical:hover { background: #10a37f; }"
            "QScrollBar::add-line:vertical { border: none; background: none; height: 0px; }"
            "QScrollBar::sub-line:vertical { border: none; background: none; height: 0px; }"
            "QScrollBar::add-page:vertical, QScrollBar::sub-page:vertical { background: none; }"
        );
        this->setAttribute(Qt::WA_StyledBackground, true);
        refreshThemedOverrides();
        return;
    }

    qApp->setStyleSheet(
        "QMainWindow { background-color: #1b1e20; }"
        "QWidget { background-color: #1b1e20; color: #eff0f1; font-family: 'Noto Sans', sans-serif; font-size: 12px; }"
        "QLabel:disabled { color: #7f8c8d; }"
        "QWidget#centralWidget { border: 1px solid #43494e; }"
        "QWidget#CustomTitleBar { background-color: #161819; border-bottom: 1px solid #43494e; }"
        "QWidget#WindowIconWidget, QLabel#WindowIconLabel { background-color: transparent; }"
        "QWidget#WindowIconWidget:hover { background-color: #2c3135; border-radius: 3px; }"
        "QDialog { border: 1px solid #43494e; }"
        "QWidget#AppDialogTitleBar { background-color: #161819; border-bottom: 1px solid #43494e; }"
        "QLabel#AppDialogTitleLabel { background-color: transparent; color: #eff0f1; font-weight: bold; }"
        "QWidget#AppDialogContent { background-color: #1b1e20; }"
        "QPushButton#AppDialogMinBtn, QPushButton#AppDialogMaxBtn { background-color: transparent; color: #eff0f1; border: none; font-size: 11px; padding: 0px; border-radius: 3px; }"
        "QPushButton#AppDialogMinBtn:hover, QPushButton#AppDialogMaxBtn:hover { background-color: #2c3135; }"
        "QPushButton#AppDialogCloseBtn { background-color: transparent; color: #eff0f1; border: none; font-size: 14px; padding: 0px; border-radius: 3px; }"
        "QPushButton#AppDialogCloseBtn:hover { background-color: #a34747; color: #ffffff; }"
        "QPushButton#TitleMinBtn, QPushButton#TitleMaxBtn { background-color: transparent; color: #eff0f1; border: none; font-size: 11px; padding: 0px; border-radius: 3px; }"
        "QPushButton#TitleMinBtn:hover, QPushButton#TitleMaxBtn:hover { background-color: #2c3135; }"
        "QPushButton#TitleCloseBtn { background-color: transparent; color: #eff0f1; border: none; font-size: 14px; padding: 0px; border-radius: 3px; }"
        "QPushButton#TitleCloseBtn:hover { background-color: #a34747; color: #ffffff; }"
        "QTabWidget::pane { border-top: 1px solid #43494e; background-color: #1b1e20; top: -1px; }"
        "QTabBar { background: transparent; outline: none; }"
        "QTabBar::tab { background-color: #23272a; color: #b0b5b9; border: 1px solid #43494e; padding: 6px 16px; border-top-left-radius: 4px; border-top-right-radius: 4px; margin-right: 2px; outline: none; }"
        "QTabBar::tab:selected { background-color: #1b1e20; color: #ffffff; border-bottom: 1px solid #1b1e20; font-weight: bold; }"
        "QTabBar::tab:hover:!selected { background-color: #2c3135; color: #eff0f1; }"
        "QTabBar::tab:focus { border: 1px solid #10a37f; color: #10a37f; }"
        "QTabBar::tab:selected:focus { border: 1px solid #10a37f; border-bottom: 1px solid #1b1e20; color: #10a37f; }"

        "QLineEdit, QTextEdit, QAbstractSpinBox { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; padding: 5px; border-radius: 4px; selection-background-color: #10a37f; selection-color: #ffffff; }"
        "QLineEdit:focus, QTextEdit:focus, QAbstractSpinBox:focus { border: 1px solid #10a37f; }"
        "QLineEdit:disabled, QTextEdit:disabled, QAbstractSpinBox:disabled { background-color: #161819; color: #7f8c8d; border: 1px solid #2c3135; }"

        "QAbstractSpinBox::up-button, QAbstractSpinBox::down-button { width: 16px; background-color: #2a2e32; border-left: 1px solid #3a3f44; }"
        "QAbstractSpinBox::up-button:hover, QAbstractSpinBox::down-button:hover { background-color: #3a3f44; }"
        "QAbstractSpinBox::up-button:pressed, QAbstractSpinBox::down-button:pressed { background-color: #10a37f; }"

        "QCalendarWidget QWidget { alternate-background-color: #2c3135; }"
        "QCalendarWidget QAbstractItemView:enabled { color: #ffffff; background-color: #23272a; selection-background-color: #10a37f; selection-color: #ffffff; outline: none; }"
        "QCalendarWidget QAbstractItemView:disabled { color: #7f8c8d; }"
        "QCalendarWidget QToolButton { color: #ffffff; background-color: transparent; border: none; margin: 2px; padding: 4px; }"
        "QCalendarWidget QToolButton:hover { background-color: #2c3135; border-radius: 3px; }"
        "QCalendarWidget QMenu { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; }"
        "QCalendarWidget QSpinBox { width: 50px; background-color: #2a2e32; border: 1px solid #3a3f44; }"

        "QComboBox { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; padding: 5px 25px 5px 5px; border-radius: 4px; combobox-popup: 1; }"
        "QComboBox:hover, QComboBox:focus { border: 1px solid #10a37f; }"
        "QComboBox:disabled { background-color: #161819; color: #7f8c8d; border: 1px solid #2c3135; }"
        "QComboBox::drop-down { subcontrol-origin: padding; subcontrol-position: top right; width: 24px; border-left: 1px solid #3a3f44; border-top-right-radius: 3px; border-bottom-right-radius: 3px; background-color: #2a2e32; }"
        "QComboBox::drop-down:hover { background-color: #3a3f44; }"
        "QComboBox::drop-down:disabled { background-color: #161819; border-left: 1px solid #2c3135; }"
        "QComboBox QAbstractItemView { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; selection-background-color: #2c3135; outline: none; }"

        "QPushButton { background-color: #23272a; color: #eff0f1; border: 1px solid #43494e; padding: 6px 12px; border-radius: 4px; font-weight: bold; }"
        "QPushButton:hover, QPushButton:focus { background-color: #10a37f; color: #ffffff; border: 1px solid #10a37f; }"
        "QPushButton:pressed { background-color: #0b7a5e; border: 1px solid #0b7a5e; }"
        "QPushButton:disabled { background-color: #161819; color: #7f8c8d; border: 1px solid #2c3135; }"

        "QCheckBox { background: transparent; color: #eff0f1; spacing: 8px; outline: none; }"
        "QCheckBox:disabled { color: #7f8c8d; }"
        "QCheckBox::indicator { width: 16px; height: 16px; border: 1px solid #3a3f44; border-radius: 3px; background-color: #23272a; }"
        "QCheckBox::indicator:hover, QCheckBox::indicator:focus { border: 1px solid #10a37f; background-color: #2c3135; }"
        "QCheckBox::indicator:checked { background-color: #10a37f; border: 1px solid #10a37f; }"
        "QCheckBox::indicator:checked:hover, QCheckBox::indicator:checked:focus { background-color: #14c79b; border: 1px solid #14c79b; }"
        "QCheckBox::indicator:disabled { background-color: #161819; border: 1px solid #2c3135; }"

        "QListWidget { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; border-radius: 4px; outline: none; }"
        "QListWidget:disabled { background-color: #161819; color: #7f8c8d; border: 1px solid #2c3135; }"
        "QListWidget::indicator { width: 16px; height: 16px; border: 1px solid #5a5a5a; border-radius: 3px; background-color: rgba(255,255,255,0.05); }"
        "QListWidget::indicator:hover, QListWidget::indicator:focus { border: 1px solid #10a37f; background-color: #2c3135; }"
        "QListWidget::indicator:checked { background-color: #10a37f; border: 1px solid #10a37f; }"
        "QListWidget::indicator:checked:hover, QListWidget::indicator:checked:focus { background-color: #14c79b; border: 1px solid #14c79b; }"
        "QListWidget::indicator:disabled { border: 1px solid #2c3135; background-color: transparent; }"

        "QMenu { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; padding: 4px; }"
        "QMenu::item { padding: 6px 24px 6px 16px; }"
        "QMenu::item:selected { background-color: #2c3135; color: #10a37f; }"
        "QScrollBar:vertical { background: #1b1e20; width: 12px; margin: 0px; }"
        "QScrollBar::handle:vertical { background: #43494e; min-height: 20px; border-radius: 6px; }"
        "QScrollBar::handle:vertical:hover { background: #10a37f; }"
        "QScrollBar::add-line:vertical { border: none; background: none; height: 0px; }"
        "QScrollBar::sub-line:vertical { border: none; background: none; height: 0px; }"
        "QScrollBar::add-page:vertical, QScrollBar::sub-page:vertical { background: none; }"

        "QComboBox QAbstractItemView QScrollBar:vertical { background: #1b1e20; width: 10px; margin: 0px; border: none; }"
        "QComboBox QAbstractItemView QScrollBar::handle:vertical { background: #43494e; min-height: 20px; border-radius: 3px; }"
        "QComboBox QAbstractItemView QScrollBar::handle:vertical:hover { background: #10a37f; }"
        "QComboBox QAbstractItemView QScrollBar::add-line:vertical, QComboBox QAbstractItemView QScrollBar::sub-line:vertical { height: 0px; background: none; }"
        "QComboBox QAbstractItemView QScrollBar::add-page:vertical, QComboBox QAbstractItemView QScrollBar::sub-page:vertical { background: #2b2b2b; }"

    );
    this->setAttribute(Qt::WA_StyledBackground, true);
    refreshThemedOverrides();
}
