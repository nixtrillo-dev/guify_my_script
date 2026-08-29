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
#include <string>
#include <utility>
#include <vector>
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
