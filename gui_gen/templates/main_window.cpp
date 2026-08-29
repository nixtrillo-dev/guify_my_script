#include <QLocalServer>
#include <QLocalSocket>
#include <functional>
#include <QApplication>
#include <QMainWindow>
#include <QAbstractItemView>
#include <QTabWidget>
#include <QTabBar>
#include <QWidget>
#include <QVBoxLayout>
#include <QHBoxLayout>
#include <QLabel>
#include <QLineEdit>
#include <QComboBox>
#include <QPushButton>
#include <QCheckBox>
#include <QFileDialog>
#include <QSystemTrayIcon>
#include <QDialog>
#include <QTreeWidget>
#include <QTreeWidgetItem>
#include <QHeaderView>
#include <QDateEdit>
#include <QTimeEdit>
#include <QSpinBox>
#include <QDoubleSpinBox>
#include <QDateTime>
#include <QCalendarWidget>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QJsonValue>
#include <QStringList>
#include <QListView>
#include <QListWidget>
#include <QListWidgetItem>
#include <QScrollBar>
#include <QStyleFactory>
#include <QCoreApplication>
#include <QDir>
#include <QMap>
#include <QMenu>
#include <QAction>
#include <QIcon>
#include <QPixmap>
#include <QProcess>
#include <QTextEdit>
#include <QClipboard>
#include <QFile>
#include <QFileInfo>
#include <QMessageBox>
#include <QFrame>
#include <QScrollArea>
#include <QPalette>
#include <QPainter>
#include <QPaintEvent>
#include <QStandardPaths>
#include <QCloseEvent>
#include <QContextMenuEvent>
#include <QEvent>
#include <QTimer>
#include <QMouseEvent>
#include <QPoint>
#include <QWindow>
#include <QScreen>
#include <QCursor>
#include <QProcessEnvironment>
#include <QRegularExpression>
#include <cstdlib>
#include <unistd.h>
#include <signal.h>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <iostream>
#include <fstream>
#include <malloc.h>
#include <QFile>
#include <QFileInfo>
#include <QProcess>
#include <QVersionNumber>
#include <QRegularExpression>

#define FALLBACK_CMD " || { echo -e \"\\n\\033[1;31m[Shell Error Handled] Press Enter to close...\\033[0m\"; read; }"

QString sourceScriptPath() {
    return TEMPLATE_SOURCE_SCRIPT_EXPR;
}
#define SOURCE_SCRIPT sourceScriptPath()
const QString APP_UUID = "TEMPLATE_UUID";
const std::string LOCK_FILE = "/tmp/cpp_qt_" + APP_UUID.toStdString() + ".lock";
const bool TRAY_ON_MINIMIZE = TEMPLATE_TRAY_ON_MINIMIZE;
const bool STARTUP_TO_TRAY = TEMPLATE_STARTUP_TO_TRAY;
const std::string INSTANCE_MODE = "TEMPLATE_INSTANCE_MODE";

int active_lock_fd = -1;
static QString g_themeColor = "dark";

static bool isLightTheme() {
    return g_themeColor.compare("light", Qt::CaseInsensitive) == 0;
}

static QString themeVal(const char *darkValue, const char *lightValue) {
    return isLightTheme() ? QString::fromLatin1(lightValue) : QString::fromLatin1(darkValue);
}

static void showWindowControlMenu(QWidget *window, const QPoint &globalPos, QPushButton *maxButton = nullptr) {
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

// Keep the combo-box affordance visible even when the platform style does not
// supply an arrow for a styled QComboBox. The glyph also reflects the popup
// state, making it clear whether the list is currently open.
class AppComboBox : public QComboBox {
public:
    using QComboBox::QComboBox;

protected:
    void showPopup() override {
        QComboBox::showPopup();
        m_popupOpen = true;
        update();
    }

    void hidePopup() override {
        m_popupOpen = false;
        update();
        QComboBox::hidePopup();
    }

    void paintEvent(QPaintEvent *event) override {
        QComboBox::paintEvent(event);

        QPainter painter(this);
        painter.setPen(QColor(!isEnabled() ? themeVal("#7f8c8d", "#8c959f") : themeVal("#ffffff", "#202124")));
        painter.drawText(QRect(width() - 25, 1, 24, height() - 2), Qt::AlignCenter,
                         m_popupOpen ? QString::fromUtf8("▲") : QString::fromUtf8("▼"));
    }

private:
    bool m_popupOpen = false;
};

// Desktop window managers own and theme ordinary dialog title bars. Use a
// small client-side title bar for generator-owned dialogs so they match the
// application's selected runtime theme on GNOME, KDE, and other desktops.
class AppDialog : public QDialog {
public:
    explicit AppDialog(QWidget *parent = nullptr) : QDialog(parent) {
        setWindowFlags(Qt::Dialog | Qt::FramelessWindowHint | Qt::WindowMinMaxButtonsHint);
        setAttribute(Qt::WA_StyledBackground, true);
        setMouseTracking(true);
        qApp->installEventFilter(this);

        QVBoxLayout *rootLayout = new QVBoxLayout(this);
        rootLayout->setContentsMargins(1, 1, 1, 1);
        rootLayout->setSpacing(0);

        QWidget *titleBar = new QWidget(this);
        titleBar->setObjectName("AppDialogTitleBar");
        titleBar->setFixedHeight(34);
        titleBar->installEventFilter(this);

        QHBoxLayout *titleLayout = new QHBoxLayout(titleBar);
        titleLayout->setContentsMargins(10, 3, 5, 3);
        titleLayout->setSpacing(6);

        m_titleLabel = new QLabel(titleBar);
        m_titleLabel->setObjectName("AppDialogTitleLabel");
        m_titleLabel->setAttribute(Qt::WA_TransparentForMouseEvents);
        titleLayout->addWidget(m_titleLabel);
        titleLayout->addStretch();

        QPushButton *minimizeButton = new QPushButton("—", titleBar);
        minimizeButton->setObjectName("AppDialogMinBtn");
        minimizeButton->setFixedSize(26, 24);
        connect(minimizeButton, &QPushButton::clicked, this, &QWidget::showMinimized);
        titleLayout->addWidget(minimizeButton);

        m_maxButton = new QPushButton("🗖", titleBar);
        m_maxButton->setObjectName("AppDialogMaxBtn");
        m_maxButton->setFixedSize(26, 24);
        connect(m_maxButton, &QPushButton::clicked, this, [this]() { toggleMaximized(); });
        titleLayout->addWidget(m_maxButton);

        QPushButton *closeButton = new QPushButton(QString::fromUtf8("✕"), titleBar);
        closeButton->setObjectName("AppDialogCloseBtn");
        closeButton->setFixedSize(26, 24);
        connect(closeButton, &QPushButton::clicked, this, &QDialog::reject);
        titleLayout->addWidget(closeButton);
        rootLayout->addWidget(titleBar);

        QWidget *content = new QWidget(this);
        content->setObjectName("AppDialogContent");
        m_contentLayout = new QVBoxLayout(content);
        rootLayout->addWidget(content, 1);
    }

    ~AppDialog() override {
        qApp->removeEventFilter(this);
    }

    QVBoxLayout *contentLayout() const {
        return m_contentLayout;
    }

protected:
    bool event(QEvent *event) override {
        const bool handled = QDialog::event(event);
        if (event->type() == QEvent::WindowTitleChange && m_titleLabel) {
            m_titleLabel->setText(windowTitle());
        } else if (event->type() == QEvent::WindowStateChange && m_maxButton) {
            m_maxButton->setText(isMaximized() ? "🗗" : "🗖");
        }
        return handled;
    }

    bool eventFilter(QObject *watched, QEvent *event) override {
        if (watched->objectName() == "AppDialogTitleBar") {
            if (event->type() == QEvent::ContextMenu) {
                QContextMenuEvent *contextEvent = static_cast<QContextMenuEvent*>(event);
                showWindowControlMenu(this, contextEvent->globalPos(), m_maxButton);
                event->accept();
                return true;
            }

            if (event->type() == QEvent::MouseButtonPress) {
                QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
                if (mouseEvent->button() == Qt::LeftButton && windowHandle()) {
                    windowHandle()->startSystemMove();
                    event->accept();
                    return true;
                }
            }
        }

        if (event->type() == QEvent::MouseMove) {
            QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
            const Qt::Edges edges = resizeEdges(mapFromGlobal(mouseEvent->globalPosition().toPoint()));
            if (edges) setCursorForEdges(edges);
            else if (cursor().shape() != Qt::ArrowCursor) setCursor(Qt::ArrowCursor);
        } else if (event->type() == QEvent::MouseButtonPress) {
            QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
            const Qt::Edges edges = resizeEdges(mapFromGlobal(mouseEvent->globalPosition().toPoint()));
            if (mouseEvent->button() == Qt::LeftButton && edges && windowHandle()) {
                windowHandle()->startSystemResize(edges);
                event->accept();
                return true;
            }
        }
        return QDialog::eventFilter(watched, event);
    }

private:
    Qt::Edges resizeEdges(const QPoint &pos) const {
        if (isMaximized()) return {};

        constexpr int border = 8;
        Qt::Edges edges = {};
        if (pos.x() <= border) edges |= Qt::LeftEdge;
        if (pos.x() >= width() - border) edges |= Qt::RightEdge;
        if (pos.y() <= border) edges |= Qt::TopEdge;
        if (pos.y() >= height() - border) edges |= Qt::BottomEdge;
        return edges;
    }

    void setCursorForEdges(Qt::Edges edges) {
        if ((edges & Qt::LeftEdge) && (edges & Qt::TopEdge)) setCursor(Qt::SizeBDiagCursor);
        else if ((edges & Qt::RightEdge) && (edges & Qt::BottomEdge)) setCursor(Qt::SizeBDiagCursor);
        else if ((edges & Qt::RightEdge) && (edges & Qt::TopEdge)) setCursor(Qt::SizeFDiagCursor);
        else if ((edges & Qt::LeftEdge) && (edges & Qt::BottomEdge)) setCursor(Qt::SizeFDiagCursor);
        else if (edges & Qt::LeftEdge || edges & Qt::RightEdge) setCursor(Qt::SizeHorCursor);
        else setCursor(Qt::SizeVerCursor);
    }

    void toggleMaximized() {
        if (isMaximized()) {
            showNormal();
            m_maxButton->setText("🗖");
        } else {
            showMaximized();
            m_maxButton->setText("🗗");
        }
    }

    QLabel *m_titleLabel = nullptr;
    QVBoxLayout *m_contentLayout = nullptr;
    QPushButton *m_maxButton = nullptr;
};

static QString appLineEditStyle() {
    return isLightTheme()
        ? "QLineEdit { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; padding: 4px; border-radius: 4px; selection-background-color: #10a37f; selection-color: #ffffff; } QLineEdit:focus { border: 1px solid #10a37f; }"
        : "QLineEdit { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; padding: 4px; border-radius: 4px; selection-background-color: #10a37f; } QLineEdit:focus { border: 1px solid #10a37f; }";
}

static QString appSubtleLabelStyle(const QString &extra = QString()) {
    return QString("color: %1; %2").arg(themeVal("#b0b5b9", "#5f6b76"), extra);
}

static QString appStrongLabelStyle(const QString &extra = QString()) {
    return QString("color: %1; font-weight: bold; background: transparent; %2").arg(themeVal("#ffffff", "#202124"), extra);
}

static QString appSeparatorStyle() {
    return QString("border: none; background-color: %1; max-height: 1px; margin-top: 5px; margin-bottom: 5px;").arg(themeVal("#43494e", "#d0d7de"));
}

static QString appLogToggleStyle() {
    return QString("QPushButton { text-align: left; background: transparent; border: none; font-weight: bold; color: %1; padding: 4px 0px; } QPushButton:hover, QPushButton:focus { color: %2; }")
        .arg(themeVal("#b0b5b9", "#5f6b76"), themeVal("#ffffff", "#202124"));
}

static QString appLogOutputStyle() {
    return isLightTheme()
        ? "font-family: 'Source Code Pro', 'Cascadia Code', 'Monospace'; background-color: #f6f8fa; color: #1f2328; border: 1px solid #d0d7de; font-size: 11px;"
        : "font-family: 'Source Code Pro', 'Cascadia Code', 'Monospace'; background-color: #161819; color: #a9b1d6; border: 1px solid #3a3f44; font-size: 11px;";
}

static QString appAccentButtonStyle(const QString &accent, const QString &hoverAccent = QString()) {
    Q_UNUSED(hoverAccent);
    return QString("QPushButton { background-color: %1; border: 1px solid %2; color: %2; padding: 8px; border-radius: 4px; font-weight: bold; } QPushButton:hover, QPushButton:focus { background-color: %2; color: #ffffff; } QPushButton:disabled { border: 1px solid %3; color: %4; background-color: %5; }")
        .arg(themeVal("#23272a", "#ffffff"), accent, themeVal("#43494e", "#d0d7de"), themeVal("#7f8c8d", "#8c959f"), themeVal("#1b1e20", "#f6f8fa"));
}

static QString appSimpleAccentButtonStyle(const QString &accent, const QString &hoverAccent = QString()) {
    const QString hover = hoverAccent.isEmpty() ? accent : hoverAccent;
    return QString("QPushButton { background-color: %1; border: 1px solid %2; color: %2; font-weight: bold; padding: 10px; border-radius: 4px; } QPushButton:hover, QPushButton:focus { background-color: %3; color: #ffffff; }")
        .arg(themeVal("#23272a", "#ffffff"), accent, hover);
}

static QString appSecondaryButtonStyle() {
    return QString("QPushButton { background-color: %1; border: 1px solid %2; color: %2; font-weight: bold; padding: 10px; border-radius: 4px; } QPushButton:hover, QPushButton:focus { background-color: %2; color: %3; }")
        .arg(themeVal("#23272a", "#ffffff"), themeVal("#b0b5b9", "#5f6b76"), themeVal("#1b1e20", "#ffffff"));
}

static QString appComboViewportStyle() {
    return QString("background-color: %1; color: %2; border: none;").arg(themeVal("#23272a", "#ffffff"), themeVal("#ffffff", "#202124"));
}

static QString appComboPopupStyle() {
    return QString(
        "QFrame { background-color: %1; border: 1px solid %2; }"
        "QComboBox { background-color: %1; color: %3; border: 1px solid %2; padding: 5px 25px 5px 5px; border-radius: 4px; combobox-popup: 1; }"
        "QComboBox:hover, QComboBox:focus { border: 1px solid #10a37f; }"
        "QComboBox::drop-down { subcontrol-origin: padding; subcontrol-position: top right; width: 24px; border-left: 1px solid %2; border-top-right-radius: 3px; border-bottom-right-radius: 3px; background-color: %4; }"
        "QComboBox::drop-down:hover { background-color: %2; }"
        "QComboBox QAbstractItemView { background-color: %1; color: %3; border: 1px solid %2; selection-background-color: %5; selection-color: %8; outline: none; }"
        "QComboBox QAbstractItemView::item:selected, QComboBox QAbstractItemView::item:hover { background-color: %5; color: %8; }"
        "QListView { background-color: %1; color: %3; border: 1px solid %2; selection-background-color: %5; selection-color: %8; outline: none; }"
        "QListView::item:selected, QListView::item:hover { background-color: %5; color: %8; }"
        "QAbstractScrollArea::corner { background: %6; border: none; }"
        "QScrollBar:vertical { background: %6; width: 10px; margin: 0px; border: none; }"
        "QScrollBar::handle:vertical { background: %2; min-height: 20px; border-radius: 3px; }"
        "QScrollBar::handle:vertical:hover { background: #10a37f; }"
        "QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical { height: 0px; background: %6; border: none; }"
        "QScrollBar::up-arrow:vertical, QScrollBar::down-arrow:vertical { background: transparent; border: none; width: 0px; height: 0px; }"
        "QScrollBar::add-page:vertical, QScrollBar::sub-page:vertical { background: %7; border: none; }"
    ).arg(themeVal("#23272a", "#ffffff"), themeVal("#3a3f44", "#c9d1d9"), themeVal("#ffffff", "#202124"), themeVal("#2a2e32", "#f6f8fa"), themeVal("#2c3135", "#cfe3ff"), themeVal("#1b1e20", "#f6f8fa"), themeVal("#2b2b2b", "#eaeef2"), themeVal("#ffffff", "#202124"));
}

static QString appMcbStyle() {
    return QString(
        "QListWidget { background-color: %1; color: %2; border: 1px solid %3; border-radius: 4px; outline: none; padding: 6px 4px 4px 4px; } "
        "QListWidget:focus { border: 1px solid #10a37f; } "
        "QListWidget::item { padding: 2px; border: 1px solid transparent; border-radius: 3px; } "
        "QListWidget::item:hover { background-color: %4; } "
        "QListWidget::item:focus, QListWidget::item:selected { background-color: rgba(16, 163, 127, 0.15); border: 1px solid #10a37f; } "
        "QListWidget QScrollBar:vertical { background: %5; width: 10px; margin: 0px; border: none; } "
        "QListWidget QScrollBar::handle:vertical { background: %6; min-height: 20px; border-radius: 3px; } "
        "QListWidget QScrollBar::handle:vertical:hover { background: #10a37f; }"
    ).arg(themeVal("#23272a", "#ffffff"), themeVal("#ffffff", "#202124"), themeVal("#3a3f44", "#c9d1d9"), themeVal("#2c3135", "#f6f8fa"), themeVal("#1b1e20", "#f6f8fa"), themeVal("#43494e", "#c9d1d9"));
}

inline QString stripAnsiEscapeCodes(const QString &text) {
    static const QRegularExpression regex("\\x1B\\[[0-9;]*[a-zA-Z]");
    QString cleaned = text;
    cleaned.remove(regex);
    return cleaned;
}

void handle_instance_policy() {
    if (INSTANCE_MODE == "duplicate") return;

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
}

// Compares semantic versions (e.g., "1.2.3" >= "1.2.0")
static bool isVersionAtLeast(const QString &installedVersion, const QString &requiredVersion) {
    QVersionNumber vInstalled = QVersionNumber::fromString(installedVersion);
    QVersionNumber vRequired = QVersionNumber::fromString(requiredVersion);
    return vInstalled >= vRequired;
}

// Queries system package manager (e.g. dpkg-query or pacman) to extract installed version
static QString getInstalledPackageVersion(const QString &pkgUUID) {
    if (pkgUUID.isEmpty()) return QString();

    QProcess proc;
    // Check via dpkg-query (Debian/Ubuntu)
    proc.start("dpkg-query", QStringList() << "-W" << "-f=${Version}" << pkgUUID);
    if (proc.waitForFinished(1000) && proc.exitCode() == 0) {
        return QString::fromUtf8(proc.readAllStandardOutput()).trimmed();
    }

    // Fallback: Check via pacman (Arch Linux)
    proc.start("pacman", QStringList() << "-Q" << pkgUUID);
    if (proc.waitForFinished(1000) && proc.exitCode() == 0) {
        QString out = QString::fromUtf8(proc.readAllStandardOutput()).trimmed();
        QStringList parts = out.split(" ");
        if (parts.size() >= 2) return parts.at(1);
    }

    return QString();
}

// Determines if an Advanced Tool tab should be made visible at runtime
static bool checkAdvancedToolEligibility(const QString &sourceScript, const QString &pkgUUID, const QString &requiredVersion) {
    // The tool is relevant only while its configured source script is present.
    // QFileInfo follows symlinks, so packaged executable links work as well.
    if (!QFileInfo(sourceScript).isFile()) {
        return false;
    }

    return true;
}

// Helper to extract nested arrays from JSON (with object array searching support)
QJsonArray getNestedArray(const QJsonDocument& doc, QString path) {
    if (path.isEmpty() || path.toLower() == "null") {
        if (doc.isArray()) return doc.array();
        return QJsonArray();
    }
    QStringList parts = path.split("/");
    QJsonValue current = doc.isArray() ? QJsonValue(doc.array()) : QJsonValue(doc.object());

    for (const QString& part : parts) {
        // Auto-unwrap single-element arrays if the path doesn't explicitly expect an array map
        if (current.isArray() && current.toArray().size() == 1 && current.toArray().at(0).isObject() && !part.startsWith('[')) {
            current = current.toArray().at(0);
        }

        if (part.startsWith('[') && part.endsWith(']')) {
            QString prop = part.mid(1, part.length() - 2);
            if (current.isArray()) {
                QJsonArray newArr;
                QJsonArray oldArr = current.toArray();
                for (const QJsonValue& val : oldArr) {
                    if (val.isObject()) {
                        QJsonValue propVal = val.toObject().value(prop);
                        if (!propVal.isUndefined() && !propVal.isNull()) {
                            newArr.append(propVal);
                        }
                    }
                }
                current = newArr;
            } else {
                return QJsonArray();
            }
        } else {
            if (current.isObject()) {
                current = current.toObject().value(part);
            } else {
                return QJsonArray();
            }
        }
    }
    return current.toArray();
}

// Helper to extract nested single or array values from JSON (for dynamic defaults)
QJsonValue getNestedValue(const QJsonDocument& doc, QString path) {
    QStringList parts = path.split("/");
    QJsonValue current = doc.isArray() ? QJsonValue(doc.array()) : QJsonValue(doc.object());

    for (const QString& part : parts) {
        // Auto-unwrap single-element arrays if the path doesn't explicitly expect an array map
        if (current.isArray() && current.toArray().size() == 1 && current.toArray().at(0).isObject() && !part.startsWith('[')) {
            current = current.toArray().at(0);
        }

        if (part.startsWith('[') && part.endsWith(']')) {
            QString prop = part.mid(1, part.length() - 2);
            if (current.isArray()) {
                QJsonArray newArr;
                QJsonArray oldArr = current.toArray();
                for (const QJsonValue& val : oldArr) {
                    if (val.isObject()) {
                        QJsonValue propVal = val.toObject().value(prop);
                        if (!propVal.isUndefined() && !propVal.isNull()) {
                            newArr.append(propVal);
                        }
                    }
                }
                current = newArr;
            } else {
                return QJsonValue();
            }
        } else {
            if (current.isObject()) {
                current = current.toObject().value(part);
            } else {
                return QJsonValue();
            }
        }
    }
    return current;
}

class InternalListWidget : public QListWidget {
protected:
    void mouseReleaseEvent(QMouseEvent *event) override {
        QListWidgetItem *itm = itemAt(event->pos());
        if (itm) {
            QRect rect = visualItemRect(itm);
            // 26px provides a safe boundary to separate the checkbox from the text.
            if (event->pos().x() > rect.x() + 26) {
                itm->setCheckState(itm->checkState() == Qt::Checked ? Qt::Unchecked : Qt::Checked);
            }
        }
        QListWidget::mouseReleaseEvent(event);
    }
public:
    InternalListWidget(QWidget *parent = nullptr) : QListWidget(parent) {}
};

class CheckableListWidget : public QWidget {
public:
    QLineEdit *searchBox;
    QWidget *headerWidget;
    QLabel *statusLabel;
    QPushButton *toggleAllBtn;
    InternalListWidget *listWidget;
    QListWidgetItem *separatorItem;

    int minChecked = 0;
    int maxChecked = 0; // 0 implies no limit
    bool isUpdating = false;

    CheckableListWidget(QWidget *parent = nullptr) : QWidget(parent) {
        QVBoxLayout *layout = new QVBoxLayout(this);
        layout->setContentsMargins(0, 0, 0, 0);
        layout->setSpacing(6);

        searchBox = new QLineEdit(this);
        searchBox->setObjectName("CheckableSearchBox");
        searchBox->setPlaceholderText("Search options...");
        searchBox->setClearButtonEnabled(true);
        searchBox->setStyleSheet(appLineEditStyle());
        searchBox->hide();

        headerWidget = new QWidget(this);
        QHBoxLayout *headerLayout = new QHBoxLayout(headerWidget);
        headerLayout->setContentsMargins(4, 0, 4, 0);
        headerLayout->setSpacing(10);

        statusLabel = new QLabel(this);
        statusLabel->setObjectName("CheckableStatusLabel");
        statusLabel->setStyleSheet(appSubtleLabelStyle("font-size: 11px; font-weight: bold;"));

        toggleAllBtn = new QPushButton(this);
        toggleAllBtn->setCursor(Qt::PointingHandCursor);
        toggleAllBtn->setStyleSheet(QString("QPushButton { background: transparent; border: none; color: #10a37f; text-decoration: underline; font-size: 11px; padding: 0; } QPushButton:hover { color: #14c79b; } QPushButton:disabled { color: %1; text-decoration: none; }").arg(themeVal("#5a6268", "#8c959f")));

        headerLayout->addWidget(statusLabel);
        headerLayout->addStretch();
        headerLayout->addWidget(toggleAllBtn);
        headerWidget->hide();

        listWidget = new InternalListWidget(this);

        // Define the visual separator
        separatorItem = new QListWidgetItem("────────── All Options ──────────");
        separatorItem->setFlags(Qt::NoItemFlags); // Disables selection/checking to act purely as a divider
        separatorItem->setTextAlignment(Qt::AlignCenter);
        listWidget->addItem(separatorItem);

        layout->addWidget(searchBox);
        layout->addWidget(headerWidget);
        layout->addWidget(listWidget);

        // Filter Logic
        connect(searchBox, &QLineEdit::textChanged, this, [this](const QString &text) {
            for (int i = 0; i < listWidget->count(); ++i) {
                QListWidgetItem *itm = listWidget->item(i);
                if (itm == separatorItem) continue;

                bool match = itm->text().contains(text, Qt::CaseInsensitive);
                itm->setHidden(!match);
            }

            // Let updateHeader handle the separator visibility entirely
            updateHeader();
        });

        // Toggle All Button Logic
        connect(toggleAllBtn, &QPushButton::clicked, this, [this]() {
            if (isUpdating) return;
            isUpdating = true;

            int sepRow = listWidget->row(separatorItem);
            int checkedCount = sepRow;
            bool doSelectAll = (maxChecked == 0 && checkedCount == 0);

            if (doSelectAll) {
                for (int i = sepRow + 1; i < listWidget->count(); ++i) {
                    QListWidgetItem *itm = listWidget->item(i);
                    if (!itm->isHidden() && itm->checkState() == Qt::Unchecked) {
                        itm->setCheckState(Qt::Checked);
                        QListWidgetItem *clone = new QListWidgetItem(itm->text());
                        clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
                        clone->setCheckState(Qt::Checked);
                        listWidget->insertItem(listWidget->row(separatorItem), clone);

                        i++; // Offset the index shift caused by the insertion
                    }
                }
            } else { // Deselect All
                // Uncheck bottom section
                for (int i = sepRow + 1; i < listWidget->count(); ++i) {
                    listWidget->item(i)->setCheckState(Qt::Unchecked);
                }
                // Purge top section
                for(int i = listWidget->row(separatorItem) - 1; i >= 0; --i) {
                    delete listWidget->takeItem(i);
                }
            }

            updateHeader();
            isUpdating = false;
        });

        // Sync Logic
        connect(listWidget, &QListWidget::itemChanged, this, [this](QListWidgetItem *item) {
            if (isUpdating || item == separatorItem) return;
            isUpdating = true;

            int sepRow = listWidget->row(separatorItem);
            int itemRow = listWidget->row(item);
            bool isTopItem = (itemRow < sepRow);
            bool checked = (item->checkState() == Qt::Checked);

            // Limit Enforcement
            if (!isTopItem && checked && maxChecked > 0) {
                int checkedCount = sepRow;
                if (checkedCount >= maxChecked) {
                    item->setCheckState(Qt::Unchecked);
                    checked = false; // Revert locally
                }
            }

            QString text = item->text();

            if (isTopItem) {
                if (!checked) {
                    // Turn off the corresponding item below the separator
                    for (int i = listWidget->row(separatorItem) + 1; i < listWidget->count(); ++i) {
                        if (listWidget->item(i)->text() == text) {
                            listWidget->item(i)->setCheckState(Qt::Unchecked);
                            break;
                        }
                    }
                    delete listWidget->takeItem(itemRow); // Remove clone
                }
            } else {
                if (checked) {
                    QListWidgetItem *clone = new QListWidgetItem(text);
                    clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
                    clone->setCheckState(Qt::Checked);
                    listWidget->insertItem(listWidget->row(separatorItem), clone);
                } else {
                    // Find and delete the clone above the separator
                    for (int i = 0; i < listWidget->row(separatorItem); ++i) {
                        if (listWidget->item(i)->text() == text) {
                            delete listWidget->takeItem(i);
                            break;
                        }
                    }
                }
            }

            updateHeader();
            isUpdating = false;
        });
    }

    void updateHeader() {
        int sepRow = listWidget->row(separatorItem);
        int checkedCount = sepRow;
        int totalItems = listWidget->count() - 1 - checkedCount; // Total minus separator minus clones

        if (maxChecked > 0) {
            statusLabel->setText(QString("%1 of %2 selected").arg(checkedCount).arg(maxChecked));
        } else {
            statusLabel->setText(QString("%1 option(s) selected").arg(checkedCount));
        }

        bool hasSearch = !searchBox->text().isEmpty();

        if (hasSearch) {
            toggleAllBtn->hide();
        } else {
            toggleAllBtn->show();
            if (maxChecked > 0) {
                toggleAllBtn->setText("Deselect All");
                toggleAllBtn->setEnabled(checkedCount > 0);
            } else {
                if (checkedCount > 0) {
                    toggleAllBtn->setText("Deselect All");
                    toggleAllBtn->setEnabled(true);
                } else {
                    toggleAllBtn->setText("Select All");
                    toggleAllBtn->setEnabled(totalItems > 0);
                }
            }
        }

        headerWidget->setVisible(totalItems > 0 || checkedCount > 0);

        if (!hasSearch) {
            separatorItem->setHidden(checkedCount == 0); // Hide separator if nothing is selected
        } else {
            // Check if there are any visible clones (selected items) matching the search
            bool hasVisibleClones = false;
            for (int i = 0; i < sepRow; ++i) {
                if (!listWidget->item(i)->isHidden()) {
                    hasVisibleClones = true;
                    break;
                }
            }
            separatorItem->setHidden(!hasVisibleClones);
        }

        if (totalItems <= 7 && !hasSearch) {
            searchBox->hide();
        } else {
            searchBox->show();
        }
    }

    // 2-argument signature for comatibility for non-aliased option values
    void addItem(const QString &text, bool checked = false) {
        QListWidgetItem *item = new QListWidgetItem(text);
        item->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
        item->setCheckState(checked ? Qt::Checked : Qt::Unchecked);
        listWidget->addItem(item); // Native insert puts this at the bottom, below the separator

        if (checked) {
            QListWidgetItem *clone = new QListWidgetItem(text);
            clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
            clone->setCheckState(Qt::Checked);
            listWidget->insertItem(listWidget->row(separatorItem), clone);
        }
        updateHeader();
    }

    // 3-argument signature to support aliased option values
    void addItem(const QString &text, const QString &userData, bool checked = false) {
        QListWidgetItem *item = new QListWidgetItem(text, listWidget);
        item->setData(Qt::UserRole, userData); // Store your underlying alias value here
        item->setFlags(item->flags() | Qt::ItemIsUserCheckable);
        item->setCheckState(checked ? Qt::Checked : Qt::Unchecked);
    }

    void clear() {
        for(int i = listWidget->count() - 1; i >= 0; --i) {
            if(listWidget->item(i) != separatorItem) {
                delete listWidget->takeItem(i); // Preserves the separator
            }
        }
        updateHeader();
    }

    QStringList checkedItems() const {
        QStringList list;
        int sepRow = listWidget->row(separatorItem);
        // We only need to check the clones above the separator
        for (int i = 0; i < sepRow; ++i) {
            list << listWidget->item(i)->text();
        }
        return list;
    }

    void setCheckedItems(const QStringList &items) {
        isUpdating = true;

        // 1. Purge clones
        int sepRow = listWidget->row(separatorItem);
        for(int i = sepRow - 1; i >= 0; --i) {
            delete listWidget->takeItem(i);
        }

        // 2. Iterate unselected base pool
        sepRow = listWidget->row(separatorItem); // Update row index after deletions
        for (int i = sepRow + 1; i < listWidget->count(); ++i) {
            QListWidgetItem *itm = listWidget->item(i);
            bool checked = items.contains(itm->text());
            itm->setCheckState(checked ? Qt::Checked : Qt::Unchecked);

            if (checked) {
                QListWidgetItem *clone = new QListWidgetItem(itm->text());
                clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
                clone->setCheckState(Qt::Checked);
                listWidget->insertItem(listWidget->row(separatorItem), clone);

                i++; // Offset the index shift caused by the insertion
            }
        }

        isUpdating = false;
        updateHeader();
    }

    bool isValid() const {
        int checkedCount = listWidget->row(separatorItem);
        if (minChecked > 0 && checkedCount < minChecked) return false;
        if (maxChecked > 0 && checkedCount > maxChecked) return false;
        return true;
    }
};

class MainWindow : public QMainWindow {
public:
    MainWindow(const QIcon &appIcon) : m_appIcon(appIcon), m_maxBtn(nullptr) {
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

    ~MainWindow() {
        cleanupAndExit();
    }

    void toggleVisibility() {
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

    Qt::Edges getResizeEdges(const QPoint &pos) {
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

protected:
    bool eventFilter(QObject *obj, QEvent *event) override {
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

    void mousePressEvent(QMouseEvent *event) override {
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

    void mouseDoubleClickEvent(QMouseEvent *event) override {
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

    void closeEvent(QCloseEvent *event) override {
        cleanupAndExit();
        event->accept();
    }

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

    // Loads JSON metadata into memory at startup
    void initMetadata() {
        QJsonObject metaObj = QJsonDocument::fromJson(QString(R"JSON(TEMPLATE_JSON_METADATA)JSON").toUtf8()).object();

        for (auto actionIt = metaObj.begin(); actionIt != metaObj.end(); ++actionIt) {
            QString actionLabel = actionIt.key();
            QJsonObject fieldsObj = actionIt.value().toObject();
            for (auto fieldIt = fieldsObj.begin(); fieldIt != fieldsObj.end(); ++fieldIt) {
                m_fieldMetadata[actionLabel + "::" + fieldIt.key()] = fieldIt.value().toObject();
            }
        }
    }

    // Helper to centralize script execution for JSON endpoints
    QJsonDocument fetchJsonEndpoint(const QStringList& endpointArgs, const QString& targetScript = SOURCE_SCRIPT) {
        if (endpointArgs.isEmpty()) return QJsonDocument();

        // ---> INTERCEPT INTERNAL ARRAYS <---
        bool isQuickLaunch = false;
        for (const QString& arg : endpointArgs) {
            if (arg.contains("__$QUICK_LAUNCH_TRIGGERS__")) {
                isQuickLaunch = true;
                break;
            }
        }

        if (isQuickLaunch) {
            QJsonArray arr;

            // Get the application path exactly as done in buildQuickLaunchRows
            QString triggerPath = QCoreApplication::applicationFilePath();
            triggerPath.replace("\"", "\\\"");

            for (const QString& key : m_quickActions.keys()) {
                QString triggerName = key;
                triggerName.replace("\"", "\\\"");

                // Format the command string exactly like the Advanced Settings UI
                QString cmd = QString("\"%1\" --trigger \"%2\"").arg(triggerPath, triggerName);

                arr.append(cmd);
            }

            QJsonDocument doc;
            doc.setArray(arr);
            return doc;
        }

        QProcess proc;
        proc.start(targetScript, endpointArgs);
        if (proc.waitForFinished(3000)) {
            return QJsonDocument::fromJson(proc.readAllStandardOutput());
        }
        return QJsonDocument();
    }

    // Overload for metadata QJsonArrays
    QJsonDocument fetchJsonEndpoint(const QJsonArray& endpointArr, const QString& actionLabel) {
        if (endpointArr.isEmpty()) return QJsonDocument();
        QStringList endpointArgs;
        for (int i = 0; i < endpointArr.size(); ++i) {
            endpointArgs.append(resolveDynamicPath(endpointArr[i].toString(), actionLabel));
        }

        // Grab the appropriate script target for this action, fallback to main app
        QString targetScript = m_actionScripts.value(actionLabel, SOURCE_SCRIPT);
        return fetchJsonEndpoint(endpointArgs, targetScript);
    }

    // Resolves raw endpoint paths using live UI values (e.g. main/{Key} -> main/f1)
    QString resolveDynamicPath(QString rawPath, const QString& actionLabel) {
        QString resolved = rawPath;
        QRegularExpression re("\\{(.+?)\\}");
        QRegularExpressionMatchIterator i = re.globalMatch(rawPath);
        while (i.hasNext()) {
            QRegularExpressionMatch match = i.next();
            QString refField = match.captured(1);
            QString mapKey = actionLabel + "::" + refField;
            QString refVal = "";
            if (m_uiFields.contains(mapKey)) {
                QWidget *w = m_uiFields[mapKey];
                if (w->property("isMCB").toBool()) refVal = static_cast<CheckableListWidget*>(w)->checkedItems().join(",");
                else if (QComboBox *cb = qobject_cast<QComboBox*>(w)) refVal = cb->currentText();
                else if (QLineEdit *le = qobject_cast<QLineEdit*>(w)) refVal = le->text();
                else if (QTextEdit *te = qobject_cast<QTextEdit*>(w)) refVal = te->toPlainText();
                else if (QDateEdit *de = qobject_cast<QDateEdit*>(w)) refVal = de->date().toString("yyyy-MM-dd");
                else if (QTimeEdit *te = qobject_cast<QTimeEdit*>(w)) refVal = te->time().toString("HH:mm:ss");
                else if (QDoubleSpinBox *dsb = qobject_cast<QDoubleSpinBox*>(w)) refVal = QString::number(dsb->value());
                else if (QSpinBox *sb = qobject_cast<QSpinBox*>(w)) refVal = QString::number(sb->value());
            }
            resolved.replace(match.captured(0), refVal);
        }
        return resolved;
    }

    // Helper to safely set widget values without infinite propagation
    void applyWidgetValue(QWidget* targetWidget, const QString& newVal) {
        QSignalBlocker blocker(targetWidget);
        if (targetWidget->property("isMCB").toBool()) {
            static_cast<CheckableListWidget*>(targetWidget)->setCheckedItems(newVal.split(",", Qt::SkipEmptyParts));
        } else if (QComboBox *cb = qobject_cast<QComboBox*>(targetWidget)) {
            if (cb->findText(newVal) == -1 && !newVal.isEmpty()) cb->addItem(newVal);
            cb->setCurrentText(newVal);
        } else if (QLineEdit *le = qobject_cast<QLineEdit*>(targetWidget)) {
            le->setText(newVal);
        } else if (QCheckBox *ck = qobject_cast<QCheckBox*>(targetWidget)) {
            ck->setChecked(newVal == "1");
        } else if (QTextEdit *te = qobject_cast<QTextEdit*>(targetWidget)) {
            te->setPlainText(newVal);
        } else if (QDateEdit *de = qobject_cast<QDateEdit*>(targetWidget)) {
            de->setDate(QDate::fromString(newVal, "yyyy-MM-dd"));
        } else if (QTimeEdit *te = qobject_cast<QTimeEdit*>(targetWidget)) {
            te->setTime(QTime::fromString(newVal, "HH:mm:ss"));
        } else if (QDoubleSpinBox *dsb = qobject_cast<QDoubleSpinBox*>(targetWidget)) {
            dsb->setValue(newVal.toDouble());
        } else if (QSpinBox *sb = qobject_cast<QSpinBox*>(targetWidget)) {
            sb->setValue(newVal.toInt());
        }
    }

    // Helper to strictly clear values to force empty output
    void clearWidgetValue(QWidget* targetWidget) {
        QSignalBlocker blocker(targetWidget);
        if (targetWidget->property("isMCB").toBool()) {
            static_cast<CheckableListWidget*>(targetWidget)->setCheckedItems(QStringList());
        } else if (QComboBox *cb = qobject_cast<QComboBox*>(targetWidget)) {
            cb->setCurrentIndex(-1);
            cb->setCurrentText("");
        } else if (QLineEdit *le = qobject_cast<QLineEdit*>(targetWidget)) {
            le->clear();
        } else if (QCheckBox *ck = qobject_cast<QCheckBox*>(targetWidget)) {
            ck->setChecked(false);
        } else if (QTextEdit *te = qobject_cast<QTextEdit*>(targetWidget)) {
            te->clear();
        } else if (QDoubleSpinBox *dsb = qobject_cast<QDoubleSpinBox*>(targetWidget)) {
            dsb->setValue(dsb->minimum());
        } else if (QSpinBox *sb = qobject_cast<QSpinBox*>(targetWidget)) {
            sb->setValue(sb->minimum());
        } else if (QDateEdit *de = qobject_cast<QDateEdit*>(targetWidget)) {
            de->setDate(de->minimumDate());
        } else if (QTimeEdit *te = qobject_cast<QTimeEdit*>(targetWidget)) {
            te->setTime(te->minimumTime());
        }
    }

    void updateWidgetOptions(QWidget* targetWidget, const QJsonObject& meta, const QJsonObject& endpoints, const QString& actionLabel) {
        QJsonArray optionsEndpointArr = endpoints["array_endpoint"].toArray();
        QString optionsPathTemplate = endpoints["path_to_array"].toString();

        if (!optionsEndpointArr.isEmpty() && (targetWidget->property("isMCB").toBool() || qobject_cast<QComboBox*>(targetWidget))) {
            QJsonDocument doc = fetchJsonEndpoint(optionsEndpointArr, actionLabel);
            if (!doc.isEmpty()) {
                QJsonArray items;

                // Intercept the internal array macro robustly
                bool isQuickLaunch = false;
                for (const QJsonValue& val : optionsEndpointArr) {
                    if (val.toString().contains("__$QUICK_LAUNCH_TRIGGERS__")) {
                        isQuickLaunch = true;
                        break;
                    }
                }

                if (isQuickLaunch) {
                    items = doc.array();
                } else {
                    QString resolvedOptionsPath = resolveDynamicPath(optionsPathTemplate, actionLabel);
                    items = getNestedArray(doc, resolvedOptionsPath);
                }

                QSignalBlocker blocker(targetWidget);
                QJsonArray staticOpts = meta["options"].toArray(); // Fetch static options

                if (targetWidget->property("isMCB").toBool()) {
                    CheckableListWidget* mcb = static_cast<CheckableListWidget*>(targetWidget);
                    mcb->clear();
                    // Add static first, then dynamic
                    for (const QJsonValue& opt : staticOpts) {
                        QString text = opt.isArray() ? opt.toArray().first().toString() : opt.toString();
                        mcb->addItem(text);
                    }
                    for (const QJsonValue& v : items) {
                        QString text = v.isArray() ? v.toArray().first().toString() : v.toString();
                        mcb->addItem(text);
                    }
                } else if (QComboBox *cb = qobject_cast<QComboBox*>(targetWidget)) {
                    cb->clear();
                    // Add static first, then dynamic
                    for (const QJsonValue& opt : staticOpts) {
                        QString text = opt.isArray() ? opt.toArray().first().toString() : opt.toString();
                        cb->addItem(text);
                    }
                    for (const QJsonValue& v : items) {
                        QString text = v.isArray() ? v.toArray().first().toString() : v.toString();
                        cb->addItem(text);
                    }
                }
            }
        }
    }

    void updateWidgetDefaultValue(QWidget* targetWidget, const QJsonObject& meta, const QJsonObject& endpoints, const QString& actionLabel, const QString& fieldName) {
        QString newVal = "";
        bool dynamicFound = false;

        QJsonArray endpointArr = endpoints["get_defaults_endpoint"].toArray();
        QString pathTemplate = endpoints["path_to_default"].toString();

        if (!endpointArr.isEmpty() && !pathTemplate.isEmpty()) {
            QJsonDocument doc = fetchJsonEndpoint(endpointArr, actionLabel);
            if (!doc.isEmpty()) {
                QString resolvedPath = resolveDynamicPath(pathTemplate, actionLabel);

                // Pass 'doc' directly instead of 'doc.object()'
                QJsonValue jVal = getNestedValue(doc, resolvedPath);

                if (!jVal.isUndefined() && !jVal.isNull()) {
                    if (jVal.isString()) newVal = jVal.toString();
                    else if (jVal.isDouble()) newVal = QString::number(jVal.toDouble());
                    else if (jVal.isBool()) newVal = jVal.toBool() ? "1" : "0";
                    else if (jVal.isArray()) {
                        QStringList list;
                        for (const QJsonValue& v : jVal.toArray()) {
                            if (v.isString()) list.append(v.toString());
                            else if (v.isDouble()) list.append(QString::number(v.toDouble()));
                            else if (v.isBool()) list.append(v.toBool() ? "1" : "0");
                        }
                        newVal = list.join(",");
                    }
                    dynamicFound = true;
                }
            }
        }

        // Hierarchy fallback if JSON fetch yields nothing
        if (!dynamicFound) {
            QString fallback = meta["default"].toString();
            newVal = getRuntimeDefault(actionLabel, fieldName, fallback);
        }

        applyWidgetValue(targetWidget, newVal);
    }

    void updateWidgetEnabledState(QWidget* targetWidget, const QJsonObject& endpoints, const QString& actionLabel) {
        QJsonArray enabledEndpointArr = endpoints["get_enabled_endpoint"].toArray();
        QString enabledPathTemplate = endpoints["path_to_enabled"].toString();

        if (!enabledEndpointArr.isEmpty() && !enabledPathTemplate.isEmpty()) {
            QJsonDocument doc = fetchJsonEndpoint(enabledEndpointArr, actionLabel);
            if (!doc.isEmpty()) {
                QString resolvedPath = resolveDynamicPath(enabledPathTemplate, actionLabel);
                QJsonValue jVal = getNestedValue(doc, resolvedPath);

                if (!jVal.isUndefined() && !jVal.isNull()) {
                    bool isEnabled = true;
                    if (jVal.isBool()) isEnabled = jVal.toBool();
                    else if (jVal.isString()) {
                        QString s = jVal.toString().toLower();
                        isEnabled = (s == "true" || s == "1");
                    }
                    else if (jVal.isDouble()) isEnabled = (jVal.toDouble() != 0);

                    targetWidget->setEnabled(isEnabled);
                    QWidget *mainLayoutItem = targetWidget;

                    // Also disable sibling widgets (like browse buttons for DIR/FL) in QHBoxLayout
                    if (targetWidget->parentWidget() && targetWidget->parentWidget()->layout()) {
                        QLayout *layout = targetWidget->parentWidget()->layout();
                        if (qobject_cast<QHBoxLayout*>(layout)) {
                            mainLayoutItem = targetWidget->parentWidget(); // Target the outer frame holding the row
                            for (int i = 0; i < layout->count(); ++i) {
                                if (QWidget *w = layout->itemAt(i)->widget()) {
                                    w->setEnabled(isEnabled);
                                }
                            }
                        }
                    }

                    // Locate the QLabel immediately preceding this field in the vertical layout
                    if (mainLayoutItem && mainLayoutItem->parentWidget() && mainLayoutItem->parentWidget()->layout()) {
                        QLayout *vLayout = mainLayoutItem->parentWidget()->layout();
                        int widgetIndex = vLayout->indexOf(mainLayoutItem);
                        if (widgetIndex > 0) {
                            if (QWidget *prevWidget = vLayout->itemAt(widgetIndex - 1)->widget()) {
                                if (qobject_cast<QLabel*>(prevWidget)) {
                                    prevWidget->setEnabled(isEnabled);
                                }
                            }
                        }
                    }

                    if (!isEnabled) {
                        clearWidgetValue(targetWidget);
                    }
                }
            }
        }
    }

    void refreshField(const QString& actionLabel, const QString& fieldName) {
        QString mapKey = actionLabel + "::" + fieldName;

        // 1. Guard clauses
        if (!m_uiFields.contains(mapKey) || !m_fieldMetadata.contains(mapKey)) {
            return;
        }

        QWidget* targetWidget = m_uiFields[mapKey];
        if (!targetWidget) {
            return;
        }

        QJsonObject meta = m_fieldMetadata[mapKey];
        QJsonObject endpoints = meta["json_endpoints"].toObject();

        // 2. Delegate execution phases
        // a. REBUILD OPTIONS DYNAMICALLY (For MCB and CB endpoints)
        updateWidgetOptions(targetWidget, meta, endpoints, actionLabel);

        // b. FETCH AND APPLY NEW DEFAULT VALUE
        updateWidgetDefaultValue(targetWidget, meta, endpoints, actionLabel, fieldName);

        // c. FETCH AND APPLY ENABLED STATE
        updateWidgetEnabledState(targetWidget, endpoints, actionLabel);
    }

    QString getRuntimeDefault(const QString& actionLabel, const QString& fieldName, const QString& fallback) {
        QString key = actionLabel + "::" + fieldName;
        if (m_runtimeDefaults.contains(key)) {
            return m_runtimeDefaults[key];
        }
        return fallback;
    }

    void initRuntimeDefaults() {
        QString configDir = QStandardPaths::writableLocation(QStandardPaths::AppConfigLocation);
        QDir().mkpath(configDir);
        QString configPath = configDir + "/runtime_defaults.json";

        QFile file(configPath);

        // 1. Load the compiled-in master template (Source of Truth)
        QJsonObject masterObj = QJsonDocument::fromJson(QString(R"JSON(TEMPLATE_JSON_DEFAULTS)JSON").toUtf8()).object();
        QJsonObject mergedObj;

        // 2. Load the user's existing config (if it exists)
        QJsonObject userObj;
        if (file.exists() && file.open(QIODevice::ReadOnly | QIODevice::Text)) {
            userObj = QJsonDocument::fromJson(file.readAll()).object();
            file.close();
        }

        // 3. Deep Merge and Prune
        for (auto it = masterObj.begin(); it != masterObj.end(); ++it) {
            QString actionLabel = it.key();
            QJsonObject masterFields = it.value().toObject();
            QJsonObject userFields;

            // Grab the user's category if it exists
            if (userObj.contains(actionLabel) && userObj[actionLabel].isObject()) {
                userFields = userObj[actionLabel].toObject();
            }

            QJsonObject mergedFields;
            for (auto fit = masterFields.begin(); fit != masterFields.end(); ++fit) {
                QString fieldName = fit.key();

                // If user has a custom value for this field, keep it.
                // Otherwise, inject the new default value.
                if (userFields.contains(fieldName)) {
                    mergedFields[fieldName] = userFields[fieldName];
                } else {
                    mergedFields[fieldName] = fit.value();
                }

                // Load the final resolved value into memory for the UI
                QString finalVal = mergedFields[fieldName].isString() ? mergedFields[fieldName].toString() : "";
                m_runtimeDefaults[actionLabel + "::" + fieldName] = finalVal;
            }
            mergedObj[actionLabel] = mergedFields;
        }

        // 4. Write the self-healed configuration back to the disk
        if (file.open(QIODevice::WriteOnly | QIODevice::Text)) {
            file.write(QJsonDocument(mergedObj).toJson(QJsonDocument::Indented));
            file.close();
        }
    }

    void cleanupAndExit() {
        try { killpg(0, SIGTERM); } catch(...) {}
        if (active_lock_fd > 0) {
            flock(active_lock_fd, LOCK_UN);
            ::close(active_lock_fd);
            unlink(LOCK_FILE.c_str());
            active_lock_fd = -1;
        }
        QApplication::quit();
    }

    void setupCustomTitleBar() {
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

    QString escapeArg(const QString &arg) {
        QString s = arg;
        s.replace("\'", "\\'").replace("\"", "\\\"").replace("$", "\\$").replace("`", "\\`");
        return s;
    }

    void initTray() {
        if (!QSystemTrayIcon::isSystemTrayAvailable()) return;

        m_trayIcon = new QSystemTrayIcon(this);
        m_trayIcon->setIcon(m_appIcon);
        m_trayIcon->setToolTip(windowTitle());

        m_trayMenu->addSeparator();
        QAction *quitAct = new QAction("Quit Toolkit", this);
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

    QString currentThemeColor() {
        QString theme = getRuntimeDefault("App Settings", "Theme Color", "TEMPLATE_THEME_COLOR").trimmed().toLower();
        if (theme == "light" || theme == "dark") return theme;
        if (theme == "auto") return detectSystemThemeColor();
        return "dark";
    }

    QString readThemeCommand(const QString &program, const QStringList &arguments, int timeoutMs = 700) const {
        if (QStandardPaths::findExecutable(program).isEmpty()) return QString();

        QProcess proc;
        proc.start(program, arguments);
        if (!proc.waitForFinished(timeoutMs) || proc.exitStatus() != QProcess::NormalExit || proc.exitCode() != 0) {
            return QString();
        }
        return QString::fromUtf8(proc.readAllStandardOutput()).trimmed().toLower();
    }

    QString detectSystemThemeColor() const {
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
    }

    void refreshThemedOverrides() {
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
                textEdit->setStyleSheet(appLogOutputStyle());
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

    void applyThemeStyle() {
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

    void initUi();
};

void MainWindow::openSettingsEditor() {
    if (m_settingsDialog) {
        m_settingsDialog->raise();
        m_settingsDialog->activateWindow();
        return;
    }

    m_settingsDialog = new AppDialog(this);
    m_settingsDialog->setWindowTitle("⚙️ Edit Runtime Defaults");
    m_settingsDialog->resize(550, 400);

    QVBoxLayout *layout = m_settingsDialog->contentLayout();

    QTreeWidget *tree = new QTreeWidget(m_settingsDialog);
    tree->setColumnCount(2);
    tree->setHeaderLabels({"Action Category / Field Name", "Default Value Override"});
    tree->header()->setSectionResizeMode(0, QHeaderView::ResizeToContents);
    tree->header()->setStretchLastSection(true);
	tree->setStyleSheet(QString(
		"QTreeWidget { background-color: %1; border: 1px solid %2; }"
		"QTreeWidget::item { padding: 4px; border-bottom: 1px solid %3; min-height: 36px; }"
		"QTreeWidget::item:selected { background-color: %7; color: %8; }"
		"QTreeWidget QAbstractScrollArea::corner { background: %1; border: none; }"
		"QTreeWidget QScrollBar:vertical { background: %1; width: 12px; margin: 0px; border: none; }"
		"QTreeWidget QScrollBar::handle:vertical { background: %2; min-height: 20px; border-radius: 6px; }"
		"QTreeWidget QScrollBar::handle:vertical:hover { background: #10a37f; }"
		"QTreeWidget QScrollBar::add-line:vertical, QTreeWidget QScrollBar::sub-line:vertical { background: %1; border: none; height: 0px; }"
		"QTreeWidget QScrollBar::up-arrow:vertical, QTreeWidget QScrollBar::down-arrow:vertical { background: transparent; border: none; width: 0px; height: 0px; }"
		"QTreeWidget QScrollBar::add-page:vertical, QTreeWidget QScrollBar::sub-page:vertical { background: %1; border: none; }"
		"QLineEdit { background: %4; color: %5; border: 1px solid %6; padding: 4px; border-radius: 3px; }"
		"QLineEdit:focus { border: 1px solid #10a37f; }"
	).arg(themeVal("#1b1e20", "#f6f8fa"), themeVal("#43494e", "#d0d7de"), themeVal("#2c3135", "#eaeef2"), themeVal("#23272a", "#ffffff"), themeVal("#ffffff", "#202124"), themeVal("#3a3f44", "#c9d1d9"), themeVal("#245b4d", "#d7f3eb"), themeVal("#ffffff", "#173b32")));
    layout->addWidget(tree);

    QString configDir = QStandardPaths::writableLocation(QStandardPaths::AppConfigLocation);
    QDir().mkpath(configDir);
    QString configPath = configDir + "/runtime_defaults.json";
    QFile file(configPath);
    QJsonObject rootObj;
    if (file.open(QIODevice::ReadOnly | QIODevice::Text)) {
        rootObj = QJsonDocument::fromJson(file.readAll()).object();
        file.close();
    }

    QJsonObject metaObj = QJsonDocument::fromJson(QString(R"JSON(TEMPLATE_JSON_METADATA)JSON").toUtf8()).object();

    for (auto it = rootObj.begin(); it != rootObj.end(); ++it) {
        QTreeWidgetItem *actionItem = new QTreeWidgetItem(tree);
        QString actionLabel = it.key();
        actionItem->setText(0, actionLabel);
        actionItem->setExpanded(true);
        actionItem->setFlags(actionItem->flags() & ~Qt::ItemIsSelectable);
        actionItem->setBackground(0, QColor(themeVal("#23272a", "#ffffff")));
        actionItem->setBackground(1, QColor(themeVal("#23272a", "#ffffff")));

        QJsonObject fieldsObj = it.value().toObject();
        QJsonObject actionMeta = metaObj[actionLabel].toObject();

        for (auto fit = fieldsObj.begin(); fit != fieldsObj.end(); ++fit) {
            QString fieldName = fit.key();
            QString val = fit.value().toString();
            QTreeWidgetItem *fieldItem = new QTreeWidgetItem(actionItem);
            fieldItem->setText(0, fieldName);

            QString fType = "TXT";
            QJsonArray fOptions;
            QJsonArray jsonEndpointArr;
            QString jsonEndpointPath = "";
            QString fMin = "";
            QString fMax = "";
            QString fDecimals = "";

            if (actionMeta.contains(fieldName)) {
                QJsonObject fMeta = actionMeta[fieldName].toObject();
                fType = fMeta["type"].toString();
                fOptions = fMeta["options"].toArray();
                fMin = fMeta["minimum"].toString();
                fMax = fMeta["maximum"].toString();
                fDecimals = fMeta["decimal_places"].toString();

                QJsonObject endpoints = fMeta["json_endpoints"].toObject();
                jsonEndpointArr = endpoints["array_endpoint"].toArray();
                jsonEndpointPath = endpoints["path_to_array"].toString();
            }

            QWidget *editorWidget = nullptr;

			const QString comboPopupStyle = appComboPopupStyle();

            if (fType == "CK") {
                QCheckBox *ck = new QCheckBox();
                ck->setChecked(val == "1");
                ck->setFocusPolicy(Qt::StrongFocus);
                editorWidget = ck;
            }
            else if (fType == "CB") {
                AppComboBox *cb = new AppComboBox();
                cb->setFocusPolicy(Qt::StrongFocus);
                cb->setMaxVisibleItems(10);

                QListView *lv = new QListView(cb);
                lv->setVerticalScrollBarPolicy(Qt::ScrollBarAlwaysOn);
                cb->setView(lv);

				cb->setStyleSheet(comboPopupStyle);
				lv->setStyleSheet(comboPopupStyle);
				lv->viewport()->setStyleSheet(appComboViewportStyle());
				lv->verticalScrollBar()->setStyleSheet(comboPopupStyle);

				if (cb->view() && cb->view()->parentWidget()) {
					cb->view()->parentWidget()->setStyleSheet(comboPopupStyle);
					cb->view()->parentWidget()->setAttribute(Qt::WA_StyledBackground, true);
					cb->view()->parentWidget()->setMaximumHeight(QGuiApplication::primaryScreen()->geometry().height() / 4);
				}

                for (int i = 0; i < fOptions.size(); ++i) {
                    QString optStr = fOptions[i].isArray() ? fOptions[i].toArray().first().toString() : fOptions[i].toString();
                    cb->addItem(optStr);
                }

                if (!jsonEndpointArr.isEmpty()) {
                    QJsonDocument doc = fetchJsonEndpoint(jsonEndpointArr, actionLabel);
                    if (!doc.isEmpty()) {
                        QJsonArray items = getNestedArray(doc, jsonEndpointPath);
                        for (const QJsonValue& v : items) {
                            QString dynOpt = v.isArray() ? v.toArray().first().toString() : v.toString();
                            cb->addItem(dynOpt);
                        }
                    }
                }
                cb->setCurrentText(val);
                editorWidget = cb;
            }
            else if (fType == "MCB") {
                CheckableListWidget *mcb = new CheckableListWidget();
                mcb->setProperty("isMCB", true);
                mcb->listWidget->setFocusPolicy(Qt::StrongFocus);
                mcb->setMaximumHeight(180);
                mcb->listWidget->setSpacing(2);

                mcb->listWidget->setStyleSheet(appMcbStyle());

                if (!fMin.isEmpty()) mcb->minChecked = fMin.toInt();
                if (!fMax.isEmpty()) mcb->maxChecked = fMax.toInt();

                for (int i = 0; i < fOptions.size(); ++i) {
                    QString optStr = fOptions[i].isArray() ? fOptions[i].toArray().first().toString() : fOptions[i].toString();
                    mcb->addItem(optStr, val.split(",", Qt::SkipEmptyParts).contains(optStr));
                }

                if (!jsonEndpointArr.isEmpty()) {
                    QJsonDocument doc = fetchJsonEndpoint(jsonEndpointArr, actionLabel);
                    if (!doc.isEmpty()) {
                        QJsonArray items = getNestedArray(doc, jsonEndpointPath);
                        for (const QJsonValue& v : items) {
                            QString dynOpt = v.isArray() ? v.toArray().first().toString() : v.toString();
                            mcb->addItem(dynOpt, val.split(",", Qt::SkipEmptyParts).contains(dynOpt));
                        }
                    }
                }
                editorWidget = mcb;
            } else if (fType == "LTXT") {
                QTextEdit *te = new QTextEdit();
                te->setPlainText(val);
                te->setAcceptRichText(false);
                te->setMaximumHeight(60);
                editorWidget = te;
            } else if (fType == "DATE") {
                QDateEdit *de = new QDateEdit();
                de->setCalendarPopup(true);
                de->setDisplayFormat("yyyy-MM-dd");
                if (!fMin.isEmpty()) de->setMinimumDate(QDate::fromString(fMin, "yyyy-MM-dd"));
                if (!fMax.isEmpty()) de->setMaximumDate(QDate::fromString(fMax, "yyyy-MM-dd"));
                de->setDate(QDate::fromString(val, "yyyy-MM-dd"));
                editorWidget = de;
            } else if (fType == "TIME") {
                QTimeEdit *te = new QTimeEdit();
                te->setDisplayFormat("HH:mm:ss");
                if (!fMin.isEmpty()) te->setMinimumTime(QTime::fromString(fMin, "HH:mm:ss"));
                if (!fMax.isEmpty()) te->setMaximumTime(QTime::fromString(fMax, "HH:mm:ss"));
                te->setTime(QTime::fromString(val, "HH:mm:ss"));
                editorWidget = te;
            } else if (fType == "NUM") {
                bool isDouble = false;
                int decimals = 6;

                if (!fDecimals.isEmpty()) {
                    if (fDecimals != "0") {
                        isDouble = true;
                        decimals = fDecimals.toInt();
                    }
                } else {
                    isDouble = val.contains('.') || fMin.contains('.') || fMax.contains('.');
                }

                if (isDouble) {
                    QDoubleSpinBox *dsb = new QDoubleSpinBox();
                    dsb->setDecimals(decimals);
                    if (!fMin.isEmpty()) dsb->setMinimum(fMin.toDouble()); else dsb->setMinimum(-999999999);
                    if (!fMax.isEmpty()) dsb->setMaximum(fMax.toDouble()); else dsb->setMaximum(999999999);
                    dsb->setValue(val.toDouble());
                    editorWidget = dsb;
                } else {
                    QSpinBox *sb = new QSpinBox();
                    if (!fMin.isEmpty()) sb->setMinimum(fMin.toInt()); else sb->setMinimum(-2147483647);
                    if (!fMax.isEmpty()) sb->setMaximum(fMax.toInt()); else sb->setMaximum(2147483647);
                    sb->setValue(val.toInt());
                    editorWidget = sb;
                }
            } else if (fType == "FD") {
                QWidget *container = new QWidget();
                QHBoxLayout *hl = new QHBoxLayout(container);
                hl->setContentsMargins(0, 0, 0, 0);

                QLineEdit *le = new QLineEdit(val);
                le->setObjectName("EditorTarget");

                QPushButton *btnFile = new QPushButton("📄 File");
                connect(btnFile, &QPushButton::clicked, [this, le]() {
                    QString f = QFileDialog::getOpenFileName(this, "Select File", le->text());
                    if(!f.isEmpty()) le->setText(f);
                });

                QPushButton *btnDir = new QPushButton("📁 Folder");
                connect(btnDir, &QPushButton::clicked, [this, le]() {
                    QString d = QFileDialog::getExistingDirectory(this, "Select Folder", le->text());
                    if(!d.isEmpty()) le->setText(d);
                });

                hl->addWidget(le);
                hl->addWidget(btnFile);
                hl->addWidget(btnDir);
                editorWidget = container;
            }
            else if (fType == "DIR" || fType == "FL" || fType == "MFL") {
                QWidget *container = new QWidget();
                QHBoxLayout *hl = new QHBoxLayout(container);

                hl->setContentsMargins(0, 0, 0, 0);

                QLineEdit *le = new QLineEdit(val);
                le->setObjectName("EditorTarget");

                QPushButton *btn = new QPushButton("Browse");
                if (fType == "DIR") {
                    connect(btn, &QPushButton::clicked, [this, le]() {
                        QString d = QFileDialog::getExistingDirectory(this, "Select Folder", le->text());
                        if(!d.isEmpty()) le->setText(d);
                    });
                } else if (fType == "FL") {
                    connect(btn, &QPushButton::clicked, [this, le]() {
                        QString f = QFileDialog::getOpenFileName(this, "Select File", le->text());
                        if(!f.isEmpty()) le->setText(f);
                    });
                } else {
                    connect(btn, &QPushButton::clicked, [this, le]() {
                        QStringList fs = QFileDialog::getOpenFileNames(this, "Select Files", le->text());
                        if(!fs.isEmpty()) le->setText(fs.join(","));
                    });
                }
                hl->addWidget(le);
                hl->addWidget(btn);
                editorWidget = container;
            }
            else {
                QLineEdit *le = new QLineEdit(val);
                editorWidget = le;
            }

            tree->setItemWidget(fieldItem, 1, editorWidget);
        }
    }

    QHBoxLayout *btnLayout = new QHBoxLayout();

    QPushButton *cancelBtn = new QPushButton("Cancel");
    cancelBtn->setStyleSheet(appSimpleAccentButtonStyle("#a34747"));
    cancelBtn->setFocusPolicy(Qt::TabFocus);

    QPushButton *saveBtn = new QPushButton("Save && Apply");
    saveBtn->setStyleSheet(appSimpleAccentButtonStyle("#10a37f"));
    saveBtn->setFocusPolicy(Qt::TabFocus);

    btnLayout->addStretch();
    btnLayout->addWidget(cancelBtn);
    btnLayout->addWidget(saveBtn);
    layout->addLayout(btnLayout);

    connect(cancelBtn, &QPushButton::clicked, m_settingsDialog, &QDialog::reject);
    connect(saveBtn, &QPushButton::clicked, [&]() {
        for (int i = 0; i < tree->topLevelItemCount(); ++i) {
            QTreeWidgetItem *actionItem = tree->topLevelItem(i);
            QString actionLabel = actionItem->text(0);
            QJsonObject fieldsObj;

            for (int j = 0; j < actionItem->childCount(); ++j) {
                QTreeWidgetItem *fieldItem = actionItem->child(j);
                QString fieldName = fieldItem->text(0);

                QWidget *w = tree->itemWidget(fieldItem, 1);
                QString newVal;

                if (w->property("isMCB").toBool()) {
                    newVal = static_cast<CheckableListWidget*>(w)->checkedItems().join(",");
                } else if (QCheckBox *ck = qobject_cast<QCheckBox*>(w)) {
                    newVal = ck->isChecked() ? "1" : "0";
                } else if (QComboBox *cb = qobject_cast<QComboBox*>(w)) {
                    newVal = cb->currentText();
                } else if (QTextEdit *te = qobject_cast<QTextEdit*>(w)) {
                    newVal = te->toPlainText();
                } else if (QDateEdit *de = qobject_cast<QDateEdit*>(w)) {
                    newVal = de->date().toString("yyyy-MM-dd");
                } else if (QTimeEdit *te = qobject_cast<QTimeEdit*>(w)) {
                    newVal = te->time().toString("HH:mm:ss");
                } else if (QDoubleSpinBox *dsb = qobject_cast<QDoubleSpinBox*>(w)) {
                    newVal = QString::number(dsb->value());
                } else if (QSpinBox *sb = qobject_cast<QSpinBox*>(w)) {
                    newVal = QString::number(sb->value());
                } else if (QLineEdit *le = w->findChild<QLineEdit*>("EditorTarget")) {
                    newVal = le->text();
                } else if (QLineEdit *le = qobject_cast<QLineEdit*>(w)) {
                    newVal = le->text();
                }

                fieldsObj[fieldName] = newVal;
                m_runtimeDefaults[actionLabel + "::" + fieldName] = newVal;

                QString mapKey = actionLabel + "::" + fieldName;
                if (m_uiFields.contains(mapKey)) {
                    QWidget *targetW = m_uiFields[mapKey];
                    if (targetW->property("isMCB").toBool()) static_cast<CheckableListWidget*>(targetW)->setCheckedItems(newVal.split(",", Qt::SkipEmptyParts));
                    else if (QLineEdit *le = qobject_cast<QLineEdit*>(targetW)) le->setText(newVal);
                    else if (QComboBox *cb = qobject_cast<QComboBox*>(targetW)) cb->setCurrentText(newVal);
                    else if (QCheckBox *ck = qobject_cast<QCheckBox*>(targetW)) ck->setChecked(newVal == "1");
                    else if (QTextEdit *te = qobject_cast<QTextEdit*>(targetW)) te->setPlainText(newVal);
                    else if (QDateEdit *de = qobject_cast<QDateEdit*>(targetW)) de->setDate(QDate::fromString(newVal, "yyyy-MM-dd"));
                    else if (QTimeEdit *te = qobject_cast<QTimeEdit*>(targetW)) te->setTime(QTime::fromString(newVal, "HH:mm:ss"));
                    else if (QDoubleSpinBox *dsb = qobject_cast<QDoubleSpinBox*>(targetW)) dsb->setValue(newVal.toDouble());
                    else if (QSpinBox *sb = qobject_cast<QSpinBox*>(targetW)) sb->setValue(newVal.toInt());
                }
            }
            rootObj[actionLabel] = fieldsObj;
        }

        if (file.open(QIODevice::WriteOnly | QIODevice::Text)) {
            file.write(QJsonDocument(rootObj).toJson(QJsonDocument::Indented));
            file.close();
        }
        applyThemeStyle();
        m_settingsDialog->accept();
    });

    m_settingsDialog->exec();
    m_settingsDialog->deleteLater();
    m_settingsDialog = nullptr;
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

    QPushButton *exitAppBtn = new QPushButton("✖ Exit Application Completely");
    exitAppBtn->setObjectName("ExitAppBtn");
    exitAppBtn->setStyleSheet(appSimpleAccentButtonStyle("#a34747"));
    exitAppBtn->setFocusPolicy(Qt::TabFocus);

    connect(exitAppBtn, &QPushButton::clicked, this, &MainWindow::cleanupAndExit);
    mainMenuLayout->addWidget(exitAppBtn);

    m_tabs->addTab(mainMenuTab, "TEMPLATE_MAIN_MENU_TAB_LABEL");

    if (qApp->arguments().contains("--settings")) {
        QTimer::singleShot(100, this, &MainWindow::openSettingsEditor);
    }
