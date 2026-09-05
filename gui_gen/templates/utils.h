// Utility functions and helpers
static bool isLightTheme() {
    return g_themeColor.compare("light", Qt::CaseInsensitive) == 0;
}

static QString themeVal(const char *darkValue, const char *lightValue) {
    return isLightTheme() ? QString::fromLatin1(lightValue) : QString::fromLatin1(darkValue);
}

static void showWindowControlMenu(QWidget *window, const QPoint &globalPos, QPushButton *maxButton = nullptr);

inline QString stripAnsiEscapeCodes(const QString &text) {
    static const QRegularExpression regex("\\x1B\\[[0-9;]*[a-zA-Z]");
    QString cleaned = text;
    cleaned.remove(regex);
    return cleaned;
}

void handle_instance_policy();

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

static QString escapePythonString(const QString& str) {
    QString esc = str;
    esc.replace('\\', "\\\\");
    esc.replace('\'', "\\'");
    esc.replace('\n', "\\n");
    esc.replace('\r', "\\r");
    esc.replace('\t', "\\t");
    return "'" + esc + "'";
}

// Quotes an argument for the bash -c command used by the generated runner.
// This keeps Python strings such as "$HOME", backticks, and quotes literal.
static QString shellQuote(const QString& str) {
    QString esc = str;
    esc.replace("'", "'\"'\"'");
    return "'" + esc + "'";
}

static QString formatPythonArg(const QString& rawVal, const QString& dataType) {
    QString dt = dataType.toLower().trimmed();
    if (dt == "int" || dt == "integer") {
        QString trimmed = rawVal.trimmed();
        if (trimmed.isEmpty()) return "0";
        bool ok = false;
        long long v = trimmed.toLongLong(&ok);
        return ok ? QString::number(v) : "0";
    } else if (dt == "float" || dt == "double") {
        QString trimmed = rawVal.trimmed();
        if (trimmed.isEmpty()) return "0.0";
        bool ok = false;
        double v = trimmed.toDouble(&ok);
        return ok ? QString::number(v) : "0.0";
    } else if (dt == "bool" || dt == "boolean") {
        QString lower = rawVal.trimmed().toLower();
        if (lower == "1" || lower == "true" || lower == "yes") return "True";
        return "False";
    } else if (dt == "list" || dt == "array") {
        if (rawVal.startsWith("[") && rawVal.endsWith("]")) return rawVal;
        QStringList parts = rawVal.split(",", Qt::SkipEmptyParts);
        QStringList formatted;
        for (const QString& p : parts) {
            formatted.append(escapePythonString(p.trimmed()));
        }
        return "[" + formatted.join(", ") + "]";
    }
    return escapePythonString(rawVal);
}

static QString jsonEscapeString(const QString& str) {
    QString escaped;
    escaped.reserve(str.size() + 2);
    for (int i = 0; i < str.size(); ++i) {
        QChar c = str.at(i);
        switch (c.unicode()) {
            case '"':  escaped += "\\\""; break;
            case '\\': escaped += "\\\\"; break;
            case '\b': escaped += "\\b";  break;
            case '\f': escaped += "\\f";  break;
            case '\n': escaped += "\\n";  break;
            case '\r': escaped += "\\r";  break;
            case '\t': escaped += "\\t";  break;
            default:
                if (c.unicode() < 0x20) {
                    escaped += QString("\\u%1").arg(c.unicode(), 4, 16, QChar('0'));
                } else {
                    escaped += c;
                }
        }
    }
    return "\"" + escaped + "\"";
}

static QString formatJsonArg(const QString& rawVal, const QString& dataType) {
    QString dt = dataType.toLower().trimmed();
    if (dt == "int" || dt == "integer") {
        QString trimmed = rawVal.trimmed();
        if (trimmed.isEmpty()) return "0";
        bool ok = false;
        long long v = trimmed.toLongLong(&ok);
        return ok ? QString::number(v) : "0";
    } else if (dt == "float" || dt == "double" || dt == "number") {
        QString trimmed = rawVal.trimmed();
        if (trimmed.isEmpty()) return "0.0";
        bool ok = false;
        double v = trimmed.toDouble(&ok);
        return ok ? QString::number(v, 'g', 15) : "0.0";
    } else if (dt == "bool" || dt == "boolean") {
        QString lower = rawVal.trimmed().toLower();
        if (lower == "1" || lower == "true" || lower == "yes") return "true";
        return "false";
    } else if (dt == "list" || dt == "array") {
        if (rawVal.trimmed().startsWith("[") && rawVal.trimmed().endsWith("]")) {
            return rawVal.trimmed();
        }
        QStringList parts = rawVal.split(",", Qt::SkipEmptyParts);
        QStringList formatted;
        for (const QString& p : parts) {
            formatted.append(jsonEscapeString(p.trimmed()));
        }
        return "[" + formatted.join(",") + "]";
    } else if (dt == "dict" || dt == "object") {
        QString trimmed = rawVal.trimmed();
        if (trimmed.startsWith("{") && trimmed.endsWith("}")) return trimmed;
        return "{}";
    }
    return jsonEscapeString(rawVal);
}

static QString formatPowerShellArg(const QString& rawVal, const QString& dataType) {
    QString dt = dataType.toLower().trimmed();
    if (dt == "bool" || dt == "boolean") {
        QString lower = rawVal.trimmed().toLower();
        return (lower == "1" || lower == "true" || lower == "yes") ? "true" : "false";
    }
    return formatJsonArg(rawVal, dataType);
}
