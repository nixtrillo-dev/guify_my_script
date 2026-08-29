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
