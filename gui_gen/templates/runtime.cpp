// Loads JSON metadata into memory at startup
void MainWindow::initMetadata() {
    QJsonObject metaObj = QJsonDocument::fromJson(QString(R"JSON(TEMPLATE_JSON_METADATA)JSON").toUtf8()).object();

    for (auto actionIt = metaObj.begin(); actionIt != metaObj.end(); ++actionIt) {
        QString actionLabel = actionIt.key();
        QJsonObject fieldsObj = actionIt.value().toObject();
        for (auto fieldIt = fieldsObj.begin(); fieldIt != fieldsObj.end(); ++fieldIt) {
            m_fieldMetadata[actionLabel + "::" + fieldIt.key()] = fieldIt.value().toObject();
        }
    }
}

QString MainWindow::getRuntimeDefault(const QString& actionLabel, const QString& fieldName, const QString& fallback) {
    QString key = actionLabel + "::" + fieldName;
    if (m_runtimeDefaults.contains(key)) {
        return m_runtimeDefaults[key];
    }
    return fallback;
}

void MainWindow::initRuntimeDefaults() {
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
