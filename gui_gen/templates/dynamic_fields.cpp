// Helper to centralize script execution for JSON endpoints
QJsonDocument MainWindow::fetchJsonEndpoint(const QStringList& endpointArgs, const QString& targetScript) {
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
QJsonDocument MainWindow::fetchJsonEndpoint(const QJsonArray& endpointArr, const QString& actionLabel) {
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
QString MainWindow::resolveDynamicPath(QString rawPath, const QString& actionLabel) {
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
            if (w->property("isMCB").toBool()) refVal = joinCsvValues(static_cast<CheckableListWidget*>(w)->checkedItems());
            else if (QComboBox *cb = qobject_cast<QComboBox*>(w)) refVal = cb->currentText();
            else if (QCheckBox *ck = qobject_cast<QCheckBox*>(w)) refVal = (!ck->isEnabled()) ? "0" : (ck->isChecked() ? "1" : "0");
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
void MainWindow::applyWidgetValue(QWidget* targetWidget, const QString& newVal) {
    QSignalBlocker blocker(targetWidget);
    if (targetWidget->property("isMCB").toBool()) {
        static_cast<CheckableListWidget*>(targetWidget)->setCheckedItems(parseCsvValues(newVal));
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
void MainWindow::clearWidgetValue(QWidget* targetWidget) {
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

void MainWindow::updateWidgetOptions(QWidget* targetWidget, const QJsonObject& meta, const QJsonObject& endpoints, const QString& actionLabel) {
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

void MainWindow::updateWidgetDefaultValue(QWidget* targetWidget, const QJsonObject& meta, const QJsonObject& endpoints, const QString& actionLabel, const QString& fieldName) {
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
                    newVal = joinCsvValues(list);
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

void MainWindow::setFieldEnabled(QWidget* targetWidget, bool isEnabled, bool clearValue) {
    if (!targetWidget) return;
    targetWidget->setEnabled(isEnabled);
    QWidget *mainLayoutItem = targetWidget;

    // Also disable/enable sibling widgets (like browse buttons for DIR/FL) in QHBoxLayout
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

    if (!isEnabled && clearValue) {
        clearWidgetValue(targetWidget);
    }
}

void MainWindow::updateWidgetEnabledState(QWidget* targetWidget, const QJsonObject& meta, const QJsonObject& endpoints, const QString& actionLabel) {
    bool isEnabled = true;
    if (meta.contains("enabled")) {
        isEnabled = meta["enabled"].toBool(true);
    }

    bool hasDynamicEndpoint = false;
    QJsonArray enabledEndpointArr = endpoints["get_enabled_endpoint"].toArray();
    QString enabledPathTemplate = endpoints["path_to_enabled"].toString();

    if (!enabledEndpointArr.isEmpty() && !enabledPathTemplate.isEmpty()) {
        QJsonDocument doc = fetchJsonEndpoint(enabledEndpointArr, actionLabel);
        if (!doc.isEmpty()) {
            QString resolvedPath = resolveDynamicPath(enabledPathTemplate, actionLabel);
            QJsonValue jVal = getNestedValue(doc, resolvedPath);

            if (!jVal.isUndefined() && !jVal.isNull()) {
                hasDynamicEndpoint = true;
                if (jVal.isBool()) isEnabled = jVal.toBool();
                else if (jVal.isString()) {
                    QString s = jVal.toString().toLower();
                    isEnabled = (s == "true" || s == "1");
                }
                else if (jVal.isDouble()) isEnabled = (jVal.toDouble() != 0);
            }
        }
    }

    setFieldEnabled(targetWidget, isEnabled, hasDynamicEndpoint && !isEnabled);
}

void MainWindow::refreshField(const QString& actionLabel, const QString& fieldName) {
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
    updateWidgetEnabledState(targetWidget, meta, endpoints, actionLabel);
}
