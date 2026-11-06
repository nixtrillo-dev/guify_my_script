static const QString ABOUT_VERSION = QStringLiteral("TEMPLATE_ABOUT_VERSION");
static const QString ABOUT_RELEASE = QStringLiteral("TEMPLATE_ABOUT_RELEASE");
static const QString ABOUT_MAINTAINER = QStringLiteral("TEMPLATE_ABOUT_MAINTAINER");
static const QString ABOUT_COMMIT = QStringLiteral("TEMPLATE_ABOUT_COMMIT");
static const QString ABOUT_SOURCE_TYPE = QStringLiteral("TEMPLATE_ABOUT_SOURCE_TYPE");
static const QString ABOUT_LICENSES_DIR = QStringLiteral("LICENSES");

class AboutLicenseComboBox final : public QComboBox {
public:
    using QComboBox::QComboBox;

protected:
    void paintEvent(QPaintEvent *event) override {
        QComboBox::paintEvent(event);

        QStyleOptionComboBox comboOption;
        initStyleOption(&comboOption);
        const QRect arrowArea = style()->subControlRect(QStyle::CC_ComboBox, &comboOption,
                                                        QStyle::SC_ComboBoxArrow, this);
        if (arrowArea.isEmpty()) return;

        QPainter painter(this);
        const QColor arrowColor(isEnabled() ? themeVal("#b0b5b9", "#5f6b76")
                                            : themeVal("#7f8c8d", "#8c959f"));
        drawAppComboArrow(&painter, arrowArea, view() && view()->isVisible(), arrowColor);
    }

    void showPopup() override {
        QComboBox::showPopup();
        if (currentIndex() >= 0 && view() && view()->selectionModel()) {
            const QModelIndex current = model()->index(currentIndex(), modelColumn());
            view()->setCurrentIndex(current);
            view()->selectionModel()->select(current,
                QItemSelectionModel::ClearAndSelect | QItemSelectionModel::Rows);
        }
        update();
    }

    void hidePopup() override {
        QComboBox::hidePopup();
        update();
    }
};

static QString queryAboutVersion(const QString &program, const QStringList &arguments) {
    QProcess process;
    process.start(program, arguments);
    if (!process.waitForStarted(1500) || !process.waitForFinished(2500)) {
        process.kill();
        process.waitForFinished(500);
        return QStringLiteral("Unknown");
    }
    QString output = QString::fromLocal8Bit(process.readAllStandardOutput());
    output += QString::fromLocal8Bit(process.readAllStandardError());
    output = output.trimmed();
    return output.isEmpty() ? QStringLiteral("Unknown") : output;
}

#ifdef Q_OS_LINUX
static QString decodeOSReleaseValue(QString value) {
    value = value.trimmed();
    if (value.size() >= 2 && value.startsWith(QLatin1Char('"')) && value.endsWith(QLatin1Char('"'))) {
        value = value.mid(1, value.size() - 2);
        QString decoded;
        for (qsizetype i = 0; i < value.size(); ++i) {
            if (value.at(i) == QLatin1Char('\\') && i + 1 < value.size()) {
                const QChar next = value.at(i + 1);
                if (next == QLatin1Char('"') || next == QLatin1Char('\\') || next == QLatin1Char('$') || next == QLatin1Char('`')) {
                    decoded.append(next);
                    ++i;
                    continue;
                }
            }
            decoded.append(value.at(i));
        }
        return decoded;
    }
    if (value.size() >= 2 && value.startsWith(QLatin1Char('\'')) && value.endsWith(QLatin1Char('\''))) {
        return value.mid(1, value.size() - 2);
    }
    return value;
}

static QMap<QString, QString> readOSRelease() {
    const QStringList paths = {QStringLiteral("/etc/os-release"), QStringLiteral("/usr/lib/os-release")};
    for (const QString &path : paths) {
        QFile file(path);
        if (!file.open(QIODevice::ReadOnly | QIODevice::Text)) {
            continue;
        }
        QMap<QString, QString> values;
        while (!file.atEnd()) {
            const QString line = QString::fromUtf8(file.readLine()).trimmed();
            if (line.isEmpty() || line.startsWith(QLatin1Char('#'))) {
                continue;
            }
            const qsizetype equals = line.indexOf(QLatin1Char('='));
            if (equals <= 0) {
                continue;
            }
            values.insert(line.left(equals).trimmed(), decodeOSReleaseValue(line.mid(equals + 1)));
        }
        return values;
    }
    return {};
}
#endif

#ifdef Q_OS_WIN
static QString windowsFullBuildNumber() {
    QSettings registry(QStringLiteral("HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion"),
                       QSettings::NativeFormat);
    QString baseBuild = registry.value(QStringLiteral("CurrentBuildNumber")).toString();
    if (baseBuild.isEmpty()) {
        baseBuild = registry.value(QStringLiteral("CurrentBuild")).toString();
    }
    const QString updateBuildRevision = registry.value(QStringLiteral("UBR")).toString();
    if (baseBuild.isEmpty() || updateBuildRevision.isEmpty()) {
        return QString();
    }
    return baseBuild + QLatin1Char('.') + updateBuildRevision;
}
#endif

void MainWindow::openAboutDialog() {
    QDialog aboutDialog(this);
    const QString appNameText = windowTitle().isEmpty() ? QStringLiteral("Unknown") : windowTitle();
    aboutDialog.setWindowTitle(QStringLiteral("About ") + appNameText);
    aboutDialog.setWindowIcon(m_appIcon);
    aboutDialog.setModal(true);

    QVBoxLayout *layout = new QVBoxLayout(&aboutDialog);
    layout->setContentsMargins(20, 18, 20, 16);
    layout->setSpacing(12);

    QLabel *appName = new QLabel(QStringLiteral("<h2>") + appNameText.toHtmlEscaped() + QStringLiteral("</h2>"), &aboutDialog);
    appName->setTextFormat(Qt::RichText);
    appName->setAlignment(Qt::AlignCenter);
    layout->addWidget(appName);

    QFormLayout *details = new QFormLayout();
    details->setFieldGrowthPolicy(QFormLayout::AllNonFixedFieldsGrow);
    auto addDetail = [details, &aboutDialog](const QString &label, const QString &value) {
        QLabel *valueLabel = new QLabel(value.isEmpty() ? QStringLiteral("Unknown") : value, &aboutDialog);
        valueLabel->setTextInteractionFlags(Qt::TextSelectableByMouse);
        valueLabel->setWordWrap(true);
        details->addRow(label + QStringLiteral(":"), valueLabel);
    };

    addDetail(QStringLiteral("Version"), ABOUT_VERSION);
    addDetail(QStringLiteral("Release"), ABOUT_RELEASE);
    addDetail(QStringLiteral("Build date (UTC)"), QStringLiteral("TEMPLATE_ABOUT_BUILD_DATE"));
    addDetail(QStringLiteral("Commit"), ABOUT_COMMIT);
    addDetail(QStringLiteral("Maintainer"), ABOUT_MAINTAINER);

    QString osName = QSysInfo::prettyProductName();
    QString osVersion = QSysInfo::productVersion();
#ifdef Q_OS_LINUX
    const QMap<QString, QString> osRelease = readOSRelease();
    osName = osRelease.value(QStringLiteral("PRETTY_NAME"));
    if (osName.isEmpty()) {
        osName = osRelease.value(QStringLiteral("NAME"));
    }
    if (osName.isEmpty()) {
        osName = QStringLiteral("Linux");
    }
    osVersion = osRelease.value(QStringLiteral("VERSION_ID"));
    if (osVersion.isEmpty()) {
        osVersion = osRelease.value(QStringLiteral("VERSION"));
    }
    if (osVersion.isEmpty()) {
        const QString buildId = osRelease.value(QStringLiteral("BUILD_ID"));
        const QString kernelVersion = QSysInfo::kernelVersion();
        if (!buildId.isEmpty()) {
            osVersion = buildId;
            if (!kernelVersion.isEmpty()) {
                osVersion += QStringLiteral(", kernel ") + kernelVersion;
            }
        } else if (!kernelVersion.isEmpty()) {
            osVersion = QStringLiteral("kernel ") + kernelVersion;
        }
    }
#else
    if (osName.isEmpty()) {
        osName = QSysInfo::productType() + QStringLiteral(" ") + QSysInfo::productVersion();
    }
#ifdef Q_OS_WIN
    const QString fullBuildNumber = windowsFullBuildNumber();
    if (!fullBuildNumber.isEmpty()) {
        if (osVersion.isEmpty()) {
            osVersion = QStringLiteral("Unknown");
        }
        osVersion += QStringLiteral(" (") + fullBuildNumber + QLatin1Char(')');
    }
#endif
#endif
    addDetail(QStringLiteral("Operating system"), osName);
    addDetail(QStringLiteral("Operating system version"), osVersion);
#ifdef Q_OS_WIN
    addDetail(QStringLiteral("Platform"), QStringLiteral("Windows"));
#elif defined(Q_OS_LINUX)
    addDetail(QStringLiteral("Platform"), QStringLiteral("Linux"));
#else
    addDetail(QStringLiteral("Platform"), QSysInfo::kernelType());
#endif
    addDetail(QStringLiteral("Qt"), QStringLiteral("Build ") + QStringLiteral("TEMPLATE_ABOUT_QT_BUILD_VERSION") +
              QStringLiteral("; runtime ") + QString::fromLatin1(qVersion()));

    QLabel *qtLicenseNotice = new QLabel(
        QStringLiteral("Qt 6 is licensed under LGPLv3. See Licenses and Notices for details."),
        &aboutDialog);
    qtLicenseNotice->setStyleSheet(QStringLiteral("QLabel { color: %1; background: transparent; }")
        .arg(isLightTheme() ? QStringLiteral("#5f6b76") : QStringLiteral("#b0b5b9")));
    layout->addWidget(qtLicenseNotice);

    if (ABOUT_SOURCE_TYPE == QStringLiteral("javascript")) {
        addDetail(QStringLiteral("Node.js"), queryAboutVersion(QStringLiteral("node"), {QStringLiteral("--version")}));
    } else if (ABOUT_SOURCE_TYPE == QStringLiteral("powershell")) {
#ifdef Q_OS_WIN
        const QString program = QStringLiteral("powershell.exe");
#else
        const QString program = QStringLiteral("pwsh");
#endif
        addDetail(QStringLiteral("PowerShell"), queryAboutVersion(program, {
            QStringLiteral("-NoLogo"), QStringLiteral("-NoProfile"), QStringLiteral("-Command"),
            QStringLiteral("$PSVersionTable.PSVersion.ToString()")
        }));
    } else if (ABOUT_SOURCE_TYPE == QStringLiteral("python")) {
#ifdef Q_OS_WIN
        const QString program = QStringLiteral("python");
#else
        const QString program = QStringLiteral("python3");
#endif
        addDetail(QStringLiteral("Python"), queryAboutVersion(program, {QStringLiteral("--version")}));
    }

    layout->addLayout(details);

    QHBoxLayout *buttons = new QHBoxLayout();
    QPushButton *licenseButton = new QPushButton(QStringLiteral("Licenses and Notices"), &aboutDialog);
    licenseButton->setStyleSheet(QStringLiteral(
        "QPushButton { color: #ffffff; background-color: #087f5b; border: 1px solid #10a37f; "
        "border-radius: 5px; padding: 7px 13px; font-weight: 600; }"
        "QPushButton:hover { color: #ffffff; background-color: #066b4e; }"
        "QPushButton:pressed { color: #ffffff; background-color: #04563f; }"
        "QPushButton:focus { border: 2px solid #8ee4ca; }"));
    connect(licenseButton, &QPushButton::clicked, &aboutDialog, [&aboutDialog]() {
        const QString licensePath = QDir(applicationInstallDirectory()).filePath(ABOUT_LICENSES_DIR);
        QDirIterator licenseIterator(licensePath, QDir::Files, QDirIterator::Subdirectories);
        QStringList licensePaths;
        while (licenseIterator.hasNext()) {
            licensePaths.append(licenseIterator.next());
        }
        std::sort(licensePaths.begin(), licensePaths.end(), [](const QString &left, const QString &right) {
            return left.compare(right, Qt::CaseInsensitive) < 0;
        });

        QDialog licenseDialog(&aboutDialog);
        licenseDialog.setWindowTitle(QStringLiteral("Licenses and Notices"));
        licenseDialog.resize(760, 560);
        QVBoxLayout licenseLayout(&licenseDialog);
        QLabel instruction(QStringLiteral("Choose a license or notice to read its full text."), &licenseDialog);
        licenseLayout.addWidget(&instruction);

        AboutLicenseComboBox licenseSelector(&licenseDialog);
        applyComboBoxPopupStyle(&licenseSelector);
        QTextEdit licenseText(&licenseDialog);
        licenseText.setReadOnly(true);
        if (licensePaths.isEmpty()) {
            instruction.setText(QStringLiteral("No license files were bundled with this application."));
            licenseSelector.setEnabled(false);
            licenseText.setEnabled(false);
        } else {
            for (const QString &path : licensePaths) {
                const QString relativePath = QDir(licensePath).relativeFilePath(path);
                QString displayName = relativePath;
                if (displayName.endsWith(QStringLiteral(".txt"), Qt::CaseInsensitive) ||
                    displayName.endsWith(QStringLiteral(".md"), Qt::CaseInsensitive)) {
                    displayName.chop(displayName.endsWith(QStringLiteral(".txt"), Qt::CaseInsensitive) ? 4 : 3);
                }
                displayName.replace(QLatin1Char('/'), QStringLiteral(" / "));
                licenseSelector.addItem(displayName, path);
            }
            if (licenseSelector.currentIndex() >= 0) {
                QAbstractItemView *licenseView = licenseSelector.view();
                licenseView->setSelectionMode(QAbstractItemView::SingleSelection);
                applyItemViewSelectionStyle(licenseView);
                const QModelIndex currentLicense = licenseSelector.model()->index(
                    licenseSelector.currentIndex(), licenseSelector.modelColumn());
                licenseView->setCurrentIndex(currentLicense);
                licenseView->selectionModel()->select(currentLicense,
                    QItemSelectionModel::ClearAndSelect | QItemSelectionModel::Rows);
            }
            auto showSelectedLicense = [&licenseSelector, &licenseText]() {
                QFile file(licenseSelector.currentData().toString());
                if (file.open(QIODevice::ReadOnly)) {
                    licenseText.setPlainText(QString::fromUtf8(file.readAll()));
                } else {
                    licenseText.setPlainText(QStringLiteral("Unable to read this license file."));
                }
            };
            QObject::connect(&licenseSelector, qOverload<int>(&QComboBox::currentIndexChanged),
                             &licenseDialog, [&showSelectedLicense](int) { showSelectedLicense(); });
            showSelectedLicense();
        }
        licenseLayout.addWidget(&licenseSelector);
        licenseLayout.addWidget(&licenseText, 1);
        QHBoxLayout licenseButtons;
        licenseButtons.addStretch();
        QPushButton closeButton(QStringLiteral("Close"), &licenseDialog);
        QObject::connect(&closeButton, &QPushButton::clicked, &licenseDialog, &QDialog::accept);
        licenseButtons.addWidget(&closeButton);
        licenseLayout.addLayout(&licenseButtons);
        licenseDialog.exec();
    });
    buttons->addWidget(licenseButton);
    buttons->addStretch();
    QPushButton *closeButton = new QPushButton(QStringLiteral("Close"), &aboutDialog);
    connect(closeButton, &QPushButton::clicked, &aboutDialog, &QDialog::accept);
    buttons->addWidget(closeButton);
    layout->addLayout(buttons);

    aboutDialog.exec();
}
