// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"fmt"
	"strings"
)

type OptionPair struct {
	Display string
	Value   string
}

func parseOptions(rawOptions []interface{}) []OptionPair {
	var parsed []OptionPair
	for _, opt := range rawOptions {
		switch v := opt.(type) {
		case string:
			// 1D Array Case: ["Option1"]
			parsed = append(parsed, OptionPair{Display: v, Value: v})
		case []interface{}:
			// 2D Array Case: [["Alias", "Value"]]
			if len(v) >= 2 {
				disp, _ := v[0].(string)
				val, _ := v[1].(string)
				parsed = append(parsed, OptionPair{Display: disp, Value: val})
			} else if len(v) == 1 {
				disp, _ := v[0].(string)
				parsed = append(parsed, OptionPair{Display: disp, Value: disp})
			}
		}
	}
	return parsed
}

// Helper to abstract C++ generation for JSON endpoint execution
func buildEndpointItemsCPP(endpoints JsonEndpoints, label string) string {
	var cpp strings.Builder
	cpp.WriteString("        QStringList endpointArgs;\n")
	for _, rawArg := range endpoints.ArrayEndpoint {
		cpp.WriteString(fmt.Sprintf("        endpointArgs.append(resolveDynamicPath(\"%s\", \"%s\"));\n", cppStringLiteral(rawArg), label))
	}
	cpp.WriteString("        QJsonDocument doc = fetchJsonEndpoint(endpointArgs);\n")

	// Intercept the internal array macro to bypass nested path resolution
	cpp.WriteString("        QJsonArray items;\n")
	cpp.WriteString("        if (endpointArgs.size() == 1 && endpointArgs[0] == \"__$QUICK_LAUNCH_TRIGGERS__\") {\n")
	cpp.WriteString("            items = doc.array();\n")
	cpp.WriteString("        } else {\n")
	cpp.WriteString(fmt.Sprintf("            items = getNestedArray(doc, resolveDynamicPath(\"%s\", \"%s\"));\n", cppStringLiteral(endpoints.PathToArray), label))
	cpp.WriteString("        }\n")

	return cpp.String()
}

// buildWidgetCPP generates the Qt C++ code for a specific UI field, including its refresh signals.
func buildWidgetCPP(field UIField, label, actIdxStr, varSuffix, targetLayout, fieldsArrName string) string {
	var cpp strings.Builder

	dataType := field.DataType
	if dataType == "" {
		dataType = "string"
	}

	isDouble := false

	cpp.WriteString(fmt.Sprintf("\n    %s->addWidget(new QLabel(\"%s:\"));\n", targetLayout, field.Name))

	// Generate the default string C++ variable once for all field types
	cpp.WriteString(fmt.Sprintf("    QString def_%[1]s = getRuntimeDefault(\"%[2]s\", \"%[3]s\", \"%[4]s\");\n", varSuffix, label, field.Name, cppStringLiteral(getDefaultString(field.Default))))

	// --- Generate the QFileDialog filter argument -------
	filterArg := `""` // Empty string defaults to All Files
	if len(field.FileTypes) > 0 {
		var exts []string
		for _, ext := range field.FileTypes {
			// Strip leading dot if accidentally provided in JSON
			cleanExt := strings.TrimPrefix(ext, ".")
			exts = append(exts, "*."+cleanExt)
		}
		// Structure: "Primary Filter (*.type1 *.type2);;Fallback Filter (*)"
		filterArg = fmt.Sprintf(`"Allowed Files (%s);;All Files (*)"`, strings.Join(exts, " "))
	}
	// -----------------------------------------------------

	switch field.GetFieldType() {
	case "TXT":
		cpp.WriteString(fmt.Sprintf("    QLineEdit *txt_%s = new QLineEdit(def_%s);\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(txt_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({txt_%s, \"TXT\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = txt_%[3]s;\n", label, field.Name, varSuffix))

	case "CB":
		cpp.WriteString(fmt.Sprintf("    AppComboBox *cb_%s = new AppComboBox();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    cb_%s->setFocusPolicy(Qt::StrongFocus);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    cb_%s->setMaxVisibleItems(10);\n", varSuffix))

		cpp.WriteString(fmt.Sprintf("    QListView *listView_%s = new QListView(cb_%s);\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("    listView_%s->setVerticalScrollBarPolicy(Qt::ScrollBarAlwaysOn);\n", varSuffix))

		// Apply viewport styling to match the runtime defaults menu logic
		cpp.WriteString(fmt.Sprintf("    listView_%s->viewport()->setStyleSheet(appComboViewportStyle());\n", varSuffix))

		cpp.WriteString(fmt.Sprintf("    cb_%s->setView(listView_%s);\n", varSuffix, varSuffix))

		cpp.WriteString(fmt.Sprintf("    if (cb_%s->view() && cb_%s->view()->parentWidget()) {\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("        cb_%s->view()->parentWidget()->setMaximumHeight(QGuiApplication::primaryScreen()->geometry().height() / 4);\n", varSuffix))

		// Force QFrame popup container styling and background attribute
		cpp.WriteString(fmt.Sprintf("        cb_%s->view()->parentWidget()->setStyleSheet(appComboPopupStyle());\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        cb_%s->view()->parentWidget()->setAttribute(Qt::WA_StyledBackground, true);\n", varSuffix))
		cpp.WriteString("    }\n")

		// Add static options FIRST
		opts := parseOptions(field.Options)
		for _, opt := range opts {
			// Qt Signature: addItem(const QString &text, const QVariant &userData)
			cpp.WriteString(fmt.Sprintf("    cb_%s->addItem(\"%s\", \"%s\");\n",
				varSuffix, cppStringLiteral(opt.Display), cppStringLiteral(opt.Value)))
		}

		// Add dynamic options sequentially with alias array support
		if len(field.JsonEndpoints.ArrayEndpoint) > 0 {
			cpp.WriteString("    {\n")
			cpp.WriteString(buildEndpointItemsCPP(field.JsonEndpoints, label))
			cpp.WriteString("        for (const QJsonValue& v : items) {\n")
			cpp.WriteString("            if (v.isArray()) {\n")
			cpp.WriteString("                QJsonArray arr = v.toArray();\n")
			cpp.WriteString("                if (arr.size() >= 2) {\n")
			cpp.WriteString(fmt.Sprintf("                    cb_%s->addItem(arr[0].toString(), arr[1].toString());\n", varSuffix))
			cpp.WriteString("                } else if (arr.size() == 1) {\n")
			cpp.WriteString(fmt.Sprintf("                    cb_%s->addItem(arr[0].toString(), arr[0].toString());\n", varSuffix))
			cpp.WriteString("                }\n")
			cpp.WriteString("            } else {\n")
			cpp.WriteString(fmt.Sprintf("                cb_%s->addItem(v.toString(), v.toString());\n", varSuffix))
			cpp.WriteString("            }\n")
			cpp.WriteString("        }\n")
			cpp.WriteString("    }\n")
		}

		cpp.WriteString(fmt.Sprintf("    if (!def_%[1]s.isEmpty()) {\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        int defIdx_%[1]s = cb_%[1]s->findData(def_%[1]s);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        if (defIdx_%[1]s != -1) cb_%[1]s->setCurrentIndex(defIdx_%[1]s);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        else cb_%[1]s->setCurrentText(def_%[1]s);\n", varSuffix))
		cpp.WriteString("    }\n")
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(cb_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({cb_%s, \"CB\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = cb_%[3]s;\n", label, field.Name, varSuffix))

	case "MCB":
		cpp.WriteString(fmt.Sprintf("    CheckableListWidget *mcb_%s = new CheckableListWidget();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    mcb_%s->setProperty(\"isMCB\", true);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    mcb_%s->setProperty(\"fieldName\", \"%s\");\n", varSuffix, cppStringLiteral(field.Name)))
		cpp.WriteString(fmt.Sprintf("    mcb_%s->listWidget->setFocusPolicy(Qt::StrongFocus);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    mcb_%s->setMaximumHeight(180);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    mcb_%s->listWidget->setSpacing(2);\n", varSuffix))

		cpp.WriteString(fmt.Sprintf("    QString mcbStyle_%s = appMcbStyle();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    mcb_%s->listWidget->setStyleSheet(mcbStyle_%s);\n", varSuffix, varSuffix))

		if field.Minimum != "" {
			cpp.WriteString(fmt.Sprintf("    mcb_%s->minChecked = %s;\n", varSuffix, field.Minimum))
		}
		if field.Maximum != "" {
			cpp.WriteString(fmt.Sprintf("    mcb_%s->maxChecked = %s;\n", varSuffix, field.Maximum))
		}

		opts := parseOptions(field.Options)
		if len(field.JsonEndpoints.ArrayEndpoint) > 0 {
			cpp.WriteString("    {\n")
			cpp.WriteString(buildEndpointItemsCPP(field.JsonEndpoints, label))
			cpp.WriteString("        for (const QJsonValue& v : items) {\n")
			cpp.WriteString("            if (v.isArray()) {\n")
			cpp.WriteString("                QJsonArray arr = v.toArray();\n")
			cpp.WriteString("                if (arr.size() >= 2) {\n")
			// Full 3-parameter support for dynamic alias options
			cpp.WriteString(fmt.Sprintf("                    mcb_%s->addItem(arr[0].toString(), arr[1].toString(), def_%[1]s.split(\",\", Qt::SkipEmptyParts).contains(arr[1].toString()));\n", varSuffix))
			cpp.WriteString("                } else if (arr.size() == 1) {\n")
			cpp.WriteString(fmt.Sprintf("                    mcb_%s->addItem(arr[0].toString(), arr[0].toString(), def_%[1]s.split(\",\", Qt::SkipEmptyParts).contains(arr[0].toString()));\n", varSuffix))
			cpp.WriteString("                }\n")
			cpp.WriteString("            } else {\n")
			cpp.WriteString(fmt.Sprintf("                mcb_%s->addItem(v.toString(), v.toString(), def_%[1]s.split(\",\", Qt::SkipEmptyParts).contains(v.toString()));\n", varSuffix))
			cpp.WriteString("            }\n")
			cpp.WriteString("        }\n")
			cpp.WriteString("    }\n")
		} else {
			for _, opt := range opts {
				cpp.WriteString(fmt.Sprintf("    mcb_%s->addItem(\"%s\", \"%s\", def_%s.split(\",\", Qt::SkipEmptyParts).contains(\"%s\"));\n",
					varSuffix, cppStringLiteral(opt.Display), cppStringLiteral(opt.Value), varSuffix, cppStringLiteral(opt.Value)))
			}
		}

		cpp.WriteString(fmt.Sprintf("    %s->addWidget(mcb_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({mcb_%s, \"MCB\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = mcb_%[3]s;\n", label, field.Name, varSuffix))

	case "CK":
		cpp.WriteString(fmt.Sprintf("    QCheckBox *ck_%s = new QCheckBox();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    ck_%s->setFocusPolicy(Qt::StrongFocus);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    ck_%s->setChecked(def_%s == \"1\");\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(ck_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({ck_%s, \"CK\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = ck_%[3]s;\n", label, field.Name, varSuffix))

	case "FD":
		cpp.WriteString("    fieldFrame = new QWidget();\n")
		cpp.WriteString("    fieldHorizontal = new QHBoxLayout(fieldFrame);\n")
		cpp.WriteString("    fieldHorizontal->setContentsMargins(0,0,0,0);\n")

		cpp.WriteString(fmt.Sprintf("    QLineEdit *fd_%s = new QLineEdit(def_%s);\n", varSuffix, varSuffix))

		cppFileBtn := fmt.Sprintf("browseFileBtn_%s", varSuffix)
		cppDirBtn := fmt.Sprintf("browseDirBtn_%s", varSuffix)

		cpp.WriteString(fmt.Sprintf("    QPushButton *%s = new QPushButton(\"📄 File\");\n", cppFileBtn))
		cpp.WriteString(fmt.Sprintf("    QPushButton *%s = new QPushButton(\"📁 Folder\");\n\n", cppDirBtn))

		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(fd_%s);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(%s);\n", cppFileBtn))
		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(%s);\n", cppDirBtn))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(fieldFrame);\n", targetLayout))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({fd_%s, \"FD\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = fd_%[3]s;\n", label, field.Name, varSuffix))

		// Connect File Button
		cpp.WriteString(fmt.Sprintf("    connect(%s, &QPushButton::clicked, [this, fd_%s]() {\n", cppFileBtn, varSuffix))
		cpp.WriteString(fmt.Sprintf("        QString f = QFileDialog::getOpenFileName(this, \"Select File\", fd_%s->text(), %s);\n", varSuffix, filterArg))
		cpp.WriteString(fmt.Sprintf("        if(!f.isEmpty()) {\n"))
		cpp.WriteString(fmt.Sprintf("            fd_%s->setText(f);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("            emit fd_%s->editingFinished();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        }\n"))
		cpp.WriteString("    });\n")

		// Connect Directory Button
		cpp.WriteString(fmt.Sprintf("    connect(%s, &QPushButton::clicked, [this, fd_%s]() {\n", cppDirBtn, varSuffix))
		cpp.WriteString(fmt.Sprintf("        QString d = QFileDialog::getExistingDirectory(this, \"Select Folder\", fd_%s->text());\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        if(!d.isEmpty()) {\n"))
		cpp.WriteString(fmt.Sprintf("            fd_%s->setText(d);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("            emit fd_%s->editingFinished();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        }\n"))
		cpp.WriteString("    });\n")

	case "DIR":
		cpp.WriteString("    fieldFrame = new QWidget();\n")
		cpp.WriteString("    fieldHorizontal = new QHBoxLayout(fieldFrame);\n")
		cpp.WriteString("    fieldHorizontal->setContentsMargins(0,0,0,0);\n")

		cpp.WriteString(fmt.Sprintf("    if(def_%s.isEmpty()) def_%s = \"/home\";\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("    QLineEdit *dir_%s = new QLineEdit(def_%s);\n", varSuffix, varSuffix))

		cppDirBtn := fmt.Sprintf("browseBtn_%s", varSuffix)
		cpp.WriteString(fmt.Sprintf("    QPushButton *%s = new QPushButton(\"Select Folder\");\n\n", cppDirBtn))

		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(dir_%s);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(%s);\n", cppDirBtn))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(fieldFrame);\n", targetLayout))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({dir_%s, \"DIR\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = dir_%[3]s;\n", label, field.Name, varSuffix))
		cpp.WriteString(fmt.Sprintf("    connect(%s, &QPushButton::clicked, [this, dir_%s]() {\n", cppDirBtn, varSuffix))
		cpp.WriteString(fmt.Sprintf("        QString d = QFileDialog::getExistingDirectory(this, \"Select Folder\", dir_%s->text());\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        if(!d.isEmpty()) {\n"))
		cpp.WriteString(fmt.Sprintf("            dir_%s->setText(d);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("            emit dir_%s->editingFinished();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("        }\n"))
		cpp.WriteString("    });\n")

	case "FL", "MFL":
		cpp.WriteString("    fieldFrame = new QWidget();\n")
		cpp.WriteString("    fieldHorizontal = new QHBoxLayout(fieldFrame);\n")
		cpp.WriteString("    fieldHorizontal->setContentsMargins(0,0,0,0);\n")

		cpp.WriteString(fmt.Sprintf("    QLineEdit *fl_%s = new QLineEdit(def_%s);\n", varSuffix, varSuffix))

		cppFlBtn := fmt.Sprintf("browseBtn_%s", varSuffix)
		cpp.WriteString(fmt.Sprintf("    QPushButton *%s = new QPushButton(\"Select File\");\n\n", cppFlBtn))

		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(fl_%s);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(%s);\n", cppFlBtn))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(fieldFrame);\n", targetLayout))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({fl_%s, \"FL\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = fl_%[3]s;\n", label, field.Name, varSuffix))
		if field.GetFieldType() == "FL" {
			cpp.WriteString(fmt.Sprintf("    connect(%s, &QPushButton::clicked, [this, fl_%s]() {\n", cppFlBtn, varSuffix))
			cpp.WriteString(fmt.Sprintf("        QString f = QFileDialog::getOpenFileName(this, \"Select File\", fl_%s->text(), %s);\n", varSuffix, filterArg))
			cpp.WriteString(fmt.Sprintf("        if(!f.isEmpty()) {\n"))
			cpp.WriteString(fmt.Sprintf("            fl_%s->setText(f);\n", varSuffix))
			cpp.WriteString(fmt.Sprintf("            emit fl_%s->editingFinished();\n", varSuffix))
			cpp.WriteString(fmt.Sprintf("        }\n"))
			cpp.WriteString("    });\n")
		} else {
			cpp.WriteString(fmt.Sprintf("    connect(%s, &QPushButton::clicked, [this, fl_%s]() {\n", cppFlBtn, varSuffix))
			cpp.WriteString(fmt.Sprintf("        QStringList fs = QFileDialog::getOpenFileNames(this, \"Select Files\", fl_%s->text(), %s);\n", varSuffix, filterArg))
			cpp.WriteString(fmt.Sprintf("        if(!fs.isEmpty()) {\n"))
			cpp.WriteString("            QStringList escaped;\n")
			cpp.WriteString("            for (const QString& item : fs) {\n")
			cpp.WriteString("                QString esc = item;\n")
			cpp.WriteString("                esc.replace(\"\\\"\", \"\\\"\\\"\");\n")
			cpp.WriteString("                escaped.append(\"\\\"\" + esc + \"\\\"\");\n")
			cpp.WriteString("            }\n")
			cpp.WriteString(fmt.Sprintf("            fl_%s->setText(escaped.join(\",\"));\n", varSuffix))
			cpp.WriteString(fmt.Sprintf("            emit fl_%s->editingFinished();\n", varSuffix))
			cpp.WriteString(fmt.Sprintf("        }\n"))
			cpp.WriteString("    });\n")
		}

	case "FS":
		cpp.WriteString("    fieldFrame = new QWidget();\n")
		cpp.WriteString("    fieldHorizontal = new QHBoxLayout(fieldFrame);\n")
		cpp.WriteString("    fieldHorizontal->setContentsMargins(0,0,0,0);\n")

		cpp.WriteString(fmt.Sprintf("    QLineEdit *fs_%s = new QLineEdit(def_%s);\n", varSuffix, varSuffix))

		cppFsBtn := fmt.Sprintf("browseBtn_%s", varSuffix)
		cpp.WriteString(fmt.Sprintf("    QPushButton *%s = new QPushButton(\"💾 Save File\");\n\n", cppFsBtn))

		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(fs_%s);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    fieldHorizontal->addWidget(%s);\n", cppFsBtn))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(fieldFrame);\n", targetLayout))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({fs_%s, \"FS\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = fs_%[3]s;\n", label, field.Name, varSuffix))

		// Connect the Save File Button
		cpp.WriteString(fmt.Sprintf("    connect(%s, &QPushButton::clicked, [this, fs_%s]() {\n", cppFsBtn, varSuffix))
		cpp.WriteString(fmt.Sprintf("        QFileDialog dialog(this, \"Save File\", fs_%s->text());\n", varSuffix))
		cpp.WriteString("        dialog.setAcceptMode(QFileDialog::AcceptSave);\n")

		if filterArg != `""` {
			cpp.WriteString(fmt.Sprintf("        dialog.setNameFilter(%s);\n", filterArg))
		}

		// Map the first element in file_types as the auto-fill default suffix
		if len(field.FileTypes) > 0 {
			cleanExt := strings.TrimPrefix(field.FileTypes[0], ".")
			cpp.WriteString(fmt.Sprintf("        dialog.setDefaultSuffix(\"%s\");\n", cleanExt))
		}

		cpp.WriteString("        if(dialog.exec()) {\n")
		cpp.WriteString("            QStringList files = dialog.selectedFiles();\n")
		cpp.WriteString("            if(!files.isEmpty()) {\n")
		cpp.WriteString(fmt.Sprintf("                fs_%s->setText(files.first());\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("                emit fs_%s->editingFinished();\n", varSuffix))
		cpp.WriteString("            }\n")
		cpp.WriteString("        }\n")
		cpp.WriteString("    });\n")

	case "LTXT":
		cpp.WriteString(fmt.Sprintf("    QTextEdit *ltxt_%s = new QTextEdit();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    ltxt_%s->setPlainText(def_%s);\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("    ltxt_%s->setAcceptRichText(false);\n", varSuffix))

		// Apply height configuration
		minH := 80
		if field.MinHeight > 0 {
			minH = field.MinHeight
		}

		if field.AutoHeight {
			cpp.WriteString(fmt.Sprintf("    ltxt_%s->setMinimumHeight(%d);\n", varSuffix, minH))

			// Auto-resize lambda: Calculates document height + margins on text change
			cpp.WriteString(fmt.Sprintf("    connect(ltxt_%s, &QTextEdit::textChanged, ltxt_%s, [ltxt_%s]() {\n", varSuffix, varSuffix, varSuffix))
			cpp.WriteString(fmt.Sprintf("        int calcHeight = ltxt_%s->document()->size().height() + 15;\n", varSuffix))
			cpp.WriteString(fmt.Sprintf("        ltxt_%s->setFixedHeight(qMax(%d, calcHeight));\n", varSuffix, minH))
			cpp.WriteString("    });\n")

			// Auto-scroll lambda: Fires on arrow keys, mouse clicks, and typing
			cpp.WriteString(fmt.Sprintf("    connect(ltxt_%s, &QTextEdit::cursorPositionChanged, ltxt_%s, [ltxt_%s, fieldsScroll_%s]() {\n", varSuffix, varSuffix, varSuffix, actIdxStr))
			cpp.WriteString(fmt.Sprintf("        QTimer::singleShot(0, ltxt_%s, [ltxt_%s, fieldsScroll_%s]() {\n", varSuffix, varSuffix, actIdxStr))
			cpp.WriteString(fmt.Sprintf("            QPoint cursorPos = ltxt_%s->mapTo(fieldsScroll_%s->widget(), ltxt_%s->cursorRect().center());\n", varSuffix, actIdxStr, varSuffix))
			cpp.WriteString(fmt.Sprintf("            fieldsScroll_%s->ensureVisible(cursorPos.x(), cursorPos.y(), 0, 15);\n", actIdxStr))
			cpp.WriteString("        });\n")
			cpp.WriteString("    });\n")

			// Fire the resize event on startup asynchronously once layout is established
			cpp.WriteString(fmt.Sprintf("    QTimer::singleShot(100, ltxt_%s, [ltxt_%s]() { emit ltxt_%s->textChanged(); });\n", varSuffix, varSuffix, varSuffix))
		} else {
			// Fallback to static height if auto_height is false
			cpp.WriteString(fmt.Sprintf("    ltxt_%s->setFixedHeight(%d);\n", varSuffix, minH))
		}

		cpp.WriteString(fmt.Sprintf("    ltxt_%s->setFocusPolicy(Qt::StrongFocus);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(ltxt_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({ltxt_%s, \"LTXT\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = ltxt_%[3]s;\n", label, field.Name, varSuffix))

	case "NUM":
		decimals := "6"
		if field.DecimalPlaces != "" {
			if field.DecimalPlaces != "0" {
				isDouble = true
				decimals = field.DecimalPlaces
			}
		} else {
			isDouble = strings.Contains(getDefaultString(field.Default), ".") || strings.Contains(field.Minimum, ".") || strings.Contains(field.Maximum, ".")
		}

		if isDouble {
			cpp.WriteString(fmt.Sprintf("    QDoubleSpinBox *num_%s = new QDoubleSpinBox();\n", varSuffix))
			cpp.WriteString(fmt.Sprintf("    num_%s->setDecimals(%s);\n", varSuffix, decimals))
			if field.Minimum != "" {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMinimum(%s);\n", varSuffix, field.Minimum))
			} else {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMinimum(-999999999.0);\n", varSuffix))
			}
			if field.Maximum != "" {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMaximum(%s);\n", varSuffix, field.Maximum))
			} else {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMaximum(999999999.0);\n", varSuffix))
			}
			cpp.WriteString(fmt.Sprintf("    num_%s->setValue(def_%s.toDouble());\n", varSuffix, varSuffix))
		} else {
			cpp.WriteString(fmt.Sprintf("    QSpinBox *num_%s = new QSpinBox();\n", varSuffix))
			if field.Minimum != "" {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMinimum(%s);\n", varSuffix, field.Minimum))
			} else {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMinimum(-2147483647);\n", varSuffix))
			}
			if field.Maximum != "" {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMaximum(%s);\n", varSuffix, field.Maximum))
			} else {
				cpp.WriteString(fmt.Sprintf("    num_%s->setMaximum(2147483647);\n", varSuffix))
			}
			cpp.WriteString(fmt.Sprintf("    num_%s->setValue(def_%s.toInt());\n", varSuffix, varSuffix))
		}

		cpp.WriteString(fmt.Sprintf("    num_%s->setFocusPolicy(Qt::StrongFocus);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(num_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({num_%s, \"NUM\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = num_%[3]s;\n", label, field.Name, varSuffix))

	case "DATE":
		cpp.WriteString(fmt.Sprintf("    QDateEdit *date_%s = new QDateEdit();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    date_%s->setCalendarPopup(true);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    date_%s->setDisplayFormat(\"yyyy-MM-dd\");\n", varSuffix))
		if field.Minimum != "" {
			cpp.WriteString(fmt.Sprintf("    date_%s->setMinimumDate(QDate::fromString(\"%s\", \"yyyy-MM-dd\"));\n", varSuffix, field.Minimum))
		}
		if field.Maximum != "" {
			cpp.WriteString(fmt.Sprintf("    date_%s->setMaximumDate(QDate::fromString(\"%s\", \"yyyy-MM-dd\"));\n", varSuffix, field.Maximum))
		}
		cpp.WriteString(fmt.Sprintf("    date_%s->setDate(QDate::fromString(def_%s, \"yyyy-MM-dd\"));\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("    date_%s->setFocusPolicy(Qt::StrongFocus);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(date_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({date_%s, \"DATE\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = date_%[3]s;\n", label, field.Name, varSuffix))

	case "TIME":
		cpp.WriteString(fmt.Sprintf("    QTimeEdit *time_%s = new QTimeEdit();\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    time_%s->setDisplayFormat(\"HH:mm:ss\");\n", varSuffix))
		if field.Minimum != "" {
			cpp.WriteString(fmt.Sprintf("    time_%s->setMinimumTime(QTime::fromString(\"%s\", \"HH:mm:ss\"));\n", varSuffix, field.Minimum))
		}
		if field.Maximum != "" {
			cpp.WriteString(fmt.Sprintf("    time_%s->setMaximumTime(QTime::fromString(\"%s\", \"HH:mm:ss\"));\n", varSuffix, field.Maximum))
		}
		cpp.WriteString(fmt.Sprintf("    time_%s->setTime(QTime::fromString(def_%s, \"HH:mm:ss\"));\n", varSuffix, varSuffix))
		cpp.WriteString(fmt.Sprintf("    time_%s->setFocusPolicy(Qt::StrongFocus);\n", varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s->addWidget(time_%s);\n", targetLayout, varSuffix))
		cpp.WriteString(fmt.Sprintf("    %s.push_back({time_%s, \"TIME\", \"%s\"});\n", fieldsArrName, varSuffix, dataType))
		cpp.WriteString(fmt.Sprintf("    m_uiFields[\"%[1]s::%[2]s\"] = time_%[3]s;\n", label, field.Name, varSuffix))
	}

	// --- DYNAMIC REFRESH SIGNAL REGISTRATION ---
	if len(field.RefreshFieldsOnChange) > 0 {
		var targetVar, signalType string
		switch field.GetFieldType() {
		case "CB":
			targetVar = fmt.Sprintf("cb_%s", varSuffix)
			signalType = "QComboBox::activated"
		case "MCB":
			targetVar = fmt.Sprintf("mcb_%s->listWidget", varSuffix)
			signalType = "QListWidget::itemChanged"
		case "TXT":
			targetVar = fmt.Sprintf("txt_%s", varSuffix)
			signalType = "QLineEdit::editingFinished"
		case "FD":
			targetVar = fmt.Sprintf("fd_%s", varSuffix)
			signalType = "QLineEdit::editingFinished"
		case "DIR":
			targetVar = fmt.Sprintf("dir_%s", varSuffix)
			signalType = "QLineEdit::editingFinished"
		case "FL", "MFL":
			targetVar = fmt.Sprintf("fl_%s", varSuffix)
			signalType = "QLineEdit::editingFinished"
		case "FS":
			targetVar = fmt.Sprintf("fs_%s", varSuffix)
			signalType = "QLineEdit::editingFinished"
		case "CK":
			targetVar = fmt.Sprintf("ck_%s", varSuffix)
			signalType = "QCheckBox::stateChanged"
		case "NUM":
			if isDouble {
				targetVar = fmt.Sprintf("num_%s", varSuffix)
				signalType = "QDoubleSpinBox::editingFinished"
			} else {
				targetVar = fmt.Sprintf("num_%s", varSuffix)
				signalType = "QSpinBox::editingFinished"
			}
		case "DATE":
			targetVar = fmt.Sprintf("date_%s", varSuffix)
			signalType = "QDateEdit::dateChanged"
		case "TIME":
			targetVar = fmt.Sprintf("time_%s", varSuffix)
			signalType = "QTimeEdit::timeChanged"
		case "LTXT":
			targetVar = fmt.Sprintf("ltxt_%s", varSuffix)
			signalType = "QTextEdit::textChanged"
		}

		if targetVar != "" {
			cpp.WriteString(fmt.Sprintf("    connect(%s, &%s, this, [this]() {\n", targetVar, signalType))
			for _, target := range field.RefreshFieldsOnChange {
				cpp.WriteString(fmt.Sprintf("        refreshField(\"%s\", \"%s\");\n", label, target))
			}
			cpp.WriteString("    });\n")
		}
	}

	return cpp.String()
}
