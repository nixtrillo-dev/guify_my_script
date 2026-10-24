package gui_gen

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func configuredAdvancedTools(actions []UIAction, sourceType string) []AdvancedToolFile {
	var tools []AdvancedToolFile
	indexes := make(map[string]int)
	for _, action := range actions {
		if !action.AdvancedMenu {
			continue
		}
		title := strings.TrimSpace(action.AdvancedMenuTabTitle)
		index, exists := indexes[title]
		if !exists {
			index = len(tools)
			indexes[title] = index
			tools = append(tools, AdvancedToolFile{
				Config: AdvancedToolConfig{
					TabTitle:      title,
					SourceType:    sourceType,
					UseMainSource: true,
				},
			})
		}
		tools[index].Actions = append(tools[index].Actions, action)
	}
	return tools
}

// buildAdvancedToolTabs generates Qt C++ code for all eligible advanced tool tabs.
func buildAdvancedToolTabs(tools []AdvancedToolFile, mainPackageUUID string) string {
	if len(tools) == 0 {
		return ""
	}

	mainPackageUUID = strings.TrimSpace(mainPackageUUID)
	var cpp strings.Builder

	cpp.WriteString("\n    // ========================================================\n")
	cpp.WriteString("    // ADVANCED TOOLS TAB CONTAINER SETUP\n")
	cpp.WriteString("    // ========================================================\n")

	cpp.WriteString("    QWidget *fieldFrame = nullptr;\n")
	cpp.WriteString("    QHBoxLayout *fieldHorizontal = nullptr;\n")

	validToolCount := 0

	for toolIdx, tool := range tools {
		toolIdxStr := strconv.Itoa(toolIdx)
		cfg := tool.Config
		toolPackageUUID := strings.TrimSpace(cfg.PackageUUID)

		// An app must not embed an advanced-tool tab for itself. Apart from
		// being redundant, that can make generated action metadata collide.
		if mainPackageUUID != "" && strings.EqualFold(toolPackageUUID, mainPackageUUID) {
			continue
		}

		cpp.WriteString(fmt.Sprintf("\n    // --- Advanced Tool: %s ---\n", cppStringLiteral(cfg.TabTitle)))
		cpp.WriteString("    {\n")

		// 1. Runtime Eligibility Check
		cpp.WriteString(fmt.Sprintf("        QString toolSourceScript = QStringLiteral(\"%s\");\n", cppStringLiteral(cfg.SourceScript)))
		cpp.WriteString(fmt.Sprintf("        QString toolPkgUUID = QStringLiteral(\"%s\");\n", cppStringLiteral(cfg.PackageUUID)))
		cpp.WriteString(fmt.Sprintf("        QString toolPkgVersion = QStringLiteral(\"%s\");\n", cppStringLiteral(cfg.PackageVersion)))
		cpp.WriteString(fmt.Sprintf("        QString toolSourceGUI = QStringLiteral(\"%s\");\n", cppStringLiteral(cfg.SourceGUI)))

		if cfg.UseMainSource {
			cpp.WriteString("        if (true) {\n")
		} else {
			cpp.WriteString("        if (checkAdvancedToolEligibility(toolSourceScript, toolPkgUUID, toolPkgVersion)) {\n")
		}

		// 2. Main Tab Container Construction
		cpp.WriteString("            QWidget *toolTab = new QWidget();\n")
		cpp.WriteString(fmt.Sprintf("            QVBoxLayout *toolMainLayout_%s = new QVBoxLayout(toolTab);\n", toolIdxStr))
		cpp.WriteString(fmt.Sprintf("            toolMainLayout_%s->setContentsMargins(10, 10, 10, 10);\n", toolIdxStr))
		cpp.WriteString(fmt.Sprintf("            toolMainLayout_%s->setSpacing(10);\n", toolIdxStr))

		// 3. Source GUI Banner / Button Header (If source_gui is defined)
		if cfg.SourceGUI != "" && !cfg.UseMainSource {
			cpp.WriteString("            if (!toolSourceGUI.isEmpty() && QFile::exists(toolSourceGUI)) {\n")
			cpp.WriteString("                QHBoxLayout *guiHeaderLayout = new QHBoxLayout();\n")
			cpp.WriteString("                QLabel *guiInfoLabel = new QLabel(\"Source GUI application is available for direct configuration:\");\n")
			cpp.WriteString("                guiInfoLabel->setObjectName(\"GuiInfoLabel\");\n")
			cpp.WriteString("                guiInfoLabel->setStyleSheet(appSubtleLabelStyle(\"font-size: 11px;\"));\n")
			cpp.WriteString("                QPushButton *launchGuiBtn = new QPushButton(\"🖥 Open Source GUI\");\n")
			cpp.WriteString("                launchGuiBtn->setStyleSheet(\"QPushButton { background-color: #10a37f; color: #ffffff; font-weight: bold; padding: 4px 10px; border-radius: 3px; } QPushButton:hover { background-color: #14c79b; }\");\n")
			cpp.WriteString("                connect(launchGuiBtn, &QPushButton::clicked, [toolSourceGUI]() {\n")
			cpp.WriteString("                    QProcess::startDetached(toolSourceGUI, QStringList());\n")
			cpp.WriteString("                });\n")
			cpp.WriteString("                guiHeaderLayout->addWidget(guiInfoLabel, 1);\n")
			cpp.WriteString("                guiHeaderLayout->addWidget(launchGuiBtn);\n")
			cpp.WriteString(fmt.Sprintf("                toolMainLayout_%s->addLayout(guiHeaderLayout);\n", toolIdxStr))
			cpp.WriteString("            }\n")
		}

		// 4. Sub-Tab Container or Action List
		cpp.WriteString("            QTabWidget *actionsTabWidget = new QTabWidget();\n")

		for actIdx, action := range tool.Actions {
			actIdxStr := fmt.Sprintf("%s_%d", toolIdxStr, actIdx)
			label := action.Label
			sanitizedLabel := sanitizeIdentifier(label)
			fieldsArrName := fmt.Sprintf("adv_fields_%s_%s", sanitizedLabel, actIdxStr)

			cpp.WriteString(fmt.Sprintf("\n            // Sub-Tab Action: %s\n", label))
			// Register this action's target script for the C++ backend
			cpp.WriteString(fmt.Sprintf("            m_actionScripts[\"%s\"] = toolSourceScript;\n", cppStringLiteral(label)))

			cpp.WriteString(fmt.Sprintf("            QWidget *actSubTab_%s = new QWidget();\n", actIdxStr))
			cpp.WriteString(fmt.Sprintf("            QVBoxLayout *tabOuterLayout_%s = new QVBoxLayout(actSubTab_%s);\n", actIdxStr, actIdxStr))
			cpp.WriteString(fmt.Sprintf("            tabOuterLayout_%s->setContentsMargins(10, 10, 10, 10);\n", actIdxStr))

			cpp.WriteString(fmt.Sprintf("            QScrollArea *fieldsScroll_%s = new QScrollArea();\n", actIdxStr))
			cpp.WriteString(fmt.Sprintf("            fieldsScroll_%s->setWidgetResizable(true);\n", actIdxStr))
			cpp.WriteString(fmt.Sprintf("            fieldsScroll_%s->setFrameShape(QFrame::NoFrame);\n", actIdxStr))

			cpp.WriteString(fmt.Sprintf("            QWidget *scrollContent_%s = new QWidget();\n", actIdxStr))
			cpp.WriteString(fmt.Sprintf("            QVBoxLayout *currentLayout_%s = new QVBoxLayout(scrollContent_%s);\n", actIdxStr, actIdxStr))
			cpp.WriteString(fmt.Sprintf("            currentLayout_%s->setSpacing(10);\n", actIdxStr))

			cpp.WriteString(fmt.Sprintf("            fieldsScroll_%s->setWidget(scrollContent_%s);\n", actIdxStr, actIdxStr))
			cpp.WriteString(fmt.Sprintf("            tabOuterLayout_%s->addWidget(fieldsScroll_%s, 1);\n", actIdxStr, actIdxStr))

			cpp.WriteString(fmt.Sprintf("            std::vector<std::tuple<QWidget*, QString, QString>> %s;\n", fieldsArrName))

			hasAdvancedFields := false
			for _, field := range action.Fields {
				if field.AdvancedMenu {
					hasAdvancedFields = true
					break
				}
			}
			if hasAdvancedFields {
				cpp.WriteString(fmt.Sprintf(`
            QPushButton *fieldAdvToggle_%[1]s = new QPushButton("▶ Advanced Settings");
            fieldAdvToggle_%[1]s->setCheckable(true);
            fieldAdvToggle_%[1]s->setStyleSheet("QPushButton { text-align: left; background: transparent; border: none; font-weight: bold; color: #10a37f; padding: 4px 0px; } QPushButton:hover, QPushButton:focus { color: #14c79b; }");
            fieldAdvToggle_%[1]s->setFocusPolicy(Qt::TabFocus);

            QWidget *fieldAdvFrame_%[1]s = new QWidget();
            QVBoxLayout *fieldAdvLayout_%[1]s = new QVBoxLayout(fieldAdvFrame_%[1]s);
            fieldAdvLayout_%[1]s->setContentsMargins(10, 0, 0, 0);
            fieldAdvLayout_%[1]s->setSpacing(10);
            fieldAdvFrame_%[1]s->setVisible(false);

            connect(fieldAdvToggle_%[1]s, &QPushButton::toggled, [fieldAdvToggle_%[1]s, fieldAdvFrame_%[1]s](bool checked) {
                fieldAdvFrame_%[1]s->setVisible(checked);
                fieldAdvToggle_%[1]s->setText(checked ? "▼ Advanced Settings" : "▶ Advanced Settings");
            });
`, actIdxStr))
			}

			// Build Action UI Fields
			for fIdx, field := range action.Fields {
				fIdxStr := strconv.Itoa(fIdx)
				varSuffix := fmt.Sprintf("adv_%s_%s_%s", sanitizedLabel, actIdxStr, fIdxStr)
				layoutName := fmt.Sprintf("currentLayout_%s", actIdxStr)
				if field.AdvancedMenu {
					layoutName = fmt.Sprintf("fieldAdvLayout_%s", actIdxStr)
				}

				widgetCode := buildWidgetCPP(field, label, actIdxStr, varSuffix, layoutName, fieldsArrName)

				if !cfg.UseMainSource {
					// Force the initial endpoint load to query the advanced tool's script
					replacement := fmt.Sprintf("fetchJsonEndpoint(endpointArgs, \"%s\")", cppStringLiteral(cfg.SourceScript))
					widgetCode = strings.ReplaceAll(widgetCode, "fetchJsonEndpoint(endpointArgs)", replacement)
				}

				cpp.WriteString(widgetCode)

				// Dynamically serialize and inject JSON metadata for runtime refreshes
				fieldJSONBytes, _ := json.Marshal(field)
				cpp.WriteString(fmt.Sprintf("            {\n                QJsonDocument fDoc = QJsonDocument::fromJson(QByteArray(\"%s\"));\n                m_fieldMetadata[\"%s::%s\"] = fDoc.object();\n            }\n", cppStringLiteral(string(fieldJSONBytes)), cppStringLiteral(label), cppStringLiteral(field.Name)))
			}

			if hasAdvancedFields {
				cpp.WriteString(fmt.Sprintf(`
            currentLayout_%[1]s->addWidget(fieldAdvToggle_%[1]s);
            currentLayout_%[1]s->addWidget(fieldAdvFrame_%[1]s);
`, actIdxStr))
			}

			cpp.WriteString(fmt.Sprintf("            currentLayout_%s->addStretch();\n", actIdxStr))

			// ---> LOG UI INJECTION (1:1 Parity with Main Window) <---
			if action.GetMode() == "g" || action.GetMode() == "gc" {
				cpp.WriteString(fmt.Sprintf(`
            // Visual Separator
            QFrame *separator_%[1]s = new QFrame();
            separator_%[1]s->setObjectName("Separator_%[1]s");
            separator_%[1]s->setFrameShape(QFrame::HLine);
            separator_%[1]s->setStyleSheet(appSeparatorStyle());
            tabOuterLayout_%[1]s->addWidget(separator_%[1]s);

            // Collapsible Toggle Button
            QPushButton *logToggle_%[1]s = new QPushButton("▶ Execution Output Logs");
            logToggle_%[1]s->setObjectName("logToggle_%[1]s");
            logToggle_%[1]s->setCheckable(true);
            logToggle_%[1]s->setStyleSheet(appLogToggleStyle());
            logToggle_%[1]s->setFocusPolicy(Qt::TabFocus);
            tabOuterLayout_%[1]s->addWidget(logToggle_%[1]s);

            // Container for the Logs
            QWidget *logFrame_%[1]s = new QWidget();
            logFrame_%[1]s->setVisible(false); // Collapsed by default
            QVBoxLayout *logFrameLayout_%[1]s = new QVBoxLayout(logFrame_%[1]s);
            logFrameLayout_%[1]s->setContentsMargins(0, 0, 0, 0);

            connect(logToggle_%[1]s, &QPushButton::toggled, [logToggle_%[1]s, logFrame_%[1]s](bool checked) {
                logFrame_%[1]s->setVisible(checked);
                logToggle_%[1]s->setText(checked ? "▼ Execution Output Logs" : "▶ Execution Output Logs");
            });

            QWidget *consoleWidgetFrame_%[1]s = new QWidget();
            QHBoxLayout *consoleControlsLayout_%[1]s = new QHBoxLayout(consoleWidgetFrame_%[1]s);
            consoleControlsLayout_%[1]s->setContentsMargins(0, 0, 0, 0);
            consoleControlsLayout_%[1]s->setSpacing(8);

            QTextEdit *logOutputWindow_%[1]s = new QTextEdit();
            logOutputWindow_%[1]s->setObjectName("LogOutputWindow_%[1]s");
            logOutputWindow_%[1]s->setReadOnly(true);
            applyLogOutputStyle(logOutputWindow_%[1]s);

            // Enforces a standard terminal scrollback limit
            logOutputWindow_%[1]s->document()->setMaximumBlockCount(10000);

            QPushButton *copyToClipboardBtn_%[1]s = new QPushButton("📋 Copy Logs");
            copyToClipboardBtn_%[1]s->setFixedWidth(100);
            copyToClipboardBtn_%[1]s->setStyleSheet("font-size: 11px; padding: 4px 8px;");

            connect(copyToClipboardBtn_%[1]s, &QPushButton::clicked, [copyToClipboardBtn_%[1]s, logOutputWindow_%[1]s]() {
                QClipboard *sysClipboard = QGuiApplication::clipboard();
                sysClipboard->setText(logOutputWindow_%[1]s->toPlainText());

                // Change text to Copied!
                copyToClipboardBtn_%[1]s->setText("Copied!");

                // Revert back to original text after 2000 milliseconds
                QTimer::singleShot(2000, copyToClipboardBtn_%[1]s, [copyToClipboardBtn_%[1]s]() {
                copyToClipboardBtn_%[1]s->setText("📋 Copy Logs");
              });
            });

            consoleControlsLayout_%[1]s->addWidget(logOutputWindow_%[1]s, 1);

            QVBoxLayout *rightControlBox_%[1]s = new QVBoxLayout();
            rightControlBox_%[1]s->setContentsMargins(0,0,0,0);
            rightControlBox_%[1]s->addWidget(copyToClipboardBtn_%[1]s);
            rightControlBox_%[1]s->addStretch();
            consoleControlsLayout_%[1]s->addLayout(rightControlBox_%[1]s);

            logFrameLayout_%[1]s->addWidget(consoleWidgetFrame_%[1]s, 1);
            tabOuterLayout_%[1]s->addWidget(logFrame_%[1]s, 1);
`, actIdxStr))
			}

			// Determine button labels with fallbacks
			execLabel := action.ExecLabel
			if execLabel == "" {
				execLabel = fmt.Sprintf("Execute %s", label)
			}
			stopLabel := action.StopLabel
			if stopLabel == "" {
				stopLabel = "Stop"
			}

			isAttached := action.GetMode() == "g" || action.GetMode() == "gc" || action.GetMode() == "s"

			// Execution Buttons
			cpp.WriteString(fmt.Sprintf(`
            QHBoxLayout *actionBtnLayout_%[1]s = new QHBoxLayout();

            QPushButton *execBtn_%[1]s = new QPushButton("%[2]s");
            execBtn_%[1]s->setObjectName("execBtn_%[1]s");
            execBtn_%[1]s->setStyleSheet(appAccentButtonStyle("#10a37f"));
            execBtn_%[1]s->setFocusPolicy(Qt::TabFocus);
            actionBtnLayout_%[1]s->addWidget(execBtn_%[1]s);
`, actIdxStr, cppStringLiteral(execLabel)))

			if isAttached {
				cpp.WriteString(fmt.Sprintf(`
            QPushButton *stopBtn_%[1]s = new QPushButton("%[2]s");
            stopBtn_%[1]s->setObjectName("stopBtn_%[1]s");
            stopBtn_%[1]s->setEnabled(false);
            stopBtn_%[1]s->setStyleSheet(appAccentButtonStyle("#a34747"));
            stopBtn_%[1]s->setFocusPolicy(Qt::TabFocus);

            actionBtnLayout_%[1]s->addWidget(stopBtn_%[1]s);
`, actIdxStr, cppStringLiteral(stopLabel)))
			}

			cpp.WriteString(fmt.Sprintf(`
            tabOuterLayout_%[1]s->addLayout(actionBtnLayout_%[1]s);
`, actIdxStr))

			// Add the completed sub-tab to the widget
			cpp.WriteString(fmt.Sprintf(`
            actionsTabWidget->addTab(actSubTab_%[1]s, "%[2]s");
`, actIdxStr, label))

			// Customize Execution Lambda targeting toolSourceScript
			if cfg.UseMainSource {
				cpp.WriteString(buildExecutionLambda(action, label, actIdxStr, fieldsArrName, cfg.SourceType))
			} else {
				cpp.WriteString(buildAdvancedExecutionLambda(action, label, actIdxStr, fieldsArrName, cfg.SourceType, cfg.SourceScript))
			}
		}

		cpp.WriteString(fmt.Sprintf("            toolMainLayout_%s->addWidget(actionsTabWidget, 1);\n", toolIdxStr))
		// Bind tab changes to the RefreshFieldsOnTabOpen logic
		cpp.WriteString("            connect(actionsTabWidget, &QTabWidget::currentChanged, this, [this, actionsTabWidget](int index) {\n")
		cpp.WriteString("                QString tabText = actionsTabWidget->tabText(index);\n")

		for _, action := range tool.Actions {
			if len(action.RefreshFieldsOnTabOpen) > 0 {
				cpp.WriteString(fmt.Sprintf("                if (tabText == \"%s\") {\n", action.Label))
				for _, fName := range action.RefreshFieldsOnTabOpen {
					cpp.WriteString(fmt.Sprintf("                    refreshField(\"%s\", \"%s\");\n", cppStringLiteral(action.Label), cppStringLiteral(fName)))
				}
				cpp.WriteString("                }\n")
			}
		}
		cpp.WriteString("            });\n")
		cpp.WriteString(fmt.Sprintf("            advancedTabs->addTab(toolTab, \"%s\");\n", cppStringLiteral(cfg.TabTitle)))
		cpp.WriteString("        }\n")
		cpp.WriteString("    }\n")

		validToolCount++
	}

	cpp.WriteString("\n    if (advancedTabs->count() > 0) {\n")
	cpp.WriteString("        advancedSettingsLayout->addWidget(advancedTabs, 1);\n") // Add stretch factor 1
	cpp.WriteString("    } else {\n")
	cpp.WriteString("        advancedTabs->deleteLater();\n")
	cpp.WriteString("    }\n")

	return cpp.String()
}

// buildAdvancedExecutionLambda generates process execution logic bound to the dynamic source script.
func buildAdvancedExecutionLambda(action UIAction, label, actIdxStr, fieldsArrName, sourceType, sourceScript string) string {
	var cpp strings.Builder

	execLabel := action.ExecLabel
	if execLabel == "" {
		execLabel = fmt.Sprintf("Execute %s", label)
	}
	inProgressLabel := action.ExecInProgressLabel
	if inProgressLabel == "" {
		inProgressLabel = execLabel
	}

	// Build the completion refresh string based on JSON arrays
	var refreshCompletionStr strings.Builder
	if len(action.RefreshFieldsOnCompletion) > 0 {
		for _, fName := range action.RefreshFieldsOnCompletion {
			refreshCompletionStr.WriteString(fmt.Sprintf("                    refreshField(\"%s\", \"%s\");\n", cppStringLiteral(label), cppStringLiteral(fName)))
		}
	}
	compStr := refreshCompletionStr.String()

	mode := action.GetMode()
	funcName := action.Func

	// Dynamically update the lambda capture list to access our UI elements!
	captureVars := "this"
	if mode == "g" || mode == "gc" {
		captureVars = fmt.Sprintf("this, logToggle_%[1]s, logOutputWindow_%[1]s, execBtn_%[1]s, stopBtn_%[1]s", actIdxStr)
	} else if mode == "s" {
		captureVars = fmt.Sprintf("this, execBtn_%[1]s, stopBtn_%[1]s", actIdxStr)
	}

	cpp.WriteString(fmt.Sprintf(`
    auto execLogic_%[1]s = [%[3]s](const QStringList& args, bool closeOnFinish = false) {
        QString escaped_args;
        for (const auto &a : args) {
            escaped_args += " \"" + escapeArg(a) + "\"";
        }
        QString targetScript = QStringLiteral("%[2]s");
`, actIdxStr, cppStringLiteral(sourceScript), captureVars))

	sudoPrefix := ""
	elevated := fmt.Sprintf("%t", action.Sudo)

	if sourceType == "go" || sourceType == "binary" || sourceType == "rust" {
		cpp.WriteString(fmt.Sprintf(`
        QString shell_cmd;
        if ("%[1]s" == "s" || "%[1]s" == "d") {
            shell_cmd = QString("%[3]s\"%%1\" %[2]s%%2").arg(targetScript, escaped_args);
        } else {
            shell_cmd = QString("%[3]s\"%%1\" %[2]s%%2%%3").arg(targetScript, escaped_args, FALLBACK_CMD);
        }
        `, mode, funcName, sudoPrefix))
	} else if sourceType == "python" {
		cpp.WriteString(fmt.Sprintf(`
        QString py_args = args.join(", ");
        QString modName = QFileInfo(targetScript).completeBaseName();
        QString pyCode = QString("import sys, os; sys.path.insert(0, os.path.dirname(%%1)); import %%2; %%2.%[2]s(%%3)").arg(escapePythonString(targetScript), modName, py_args);
        QString quotedPyCode = shellQuote(pyCode);

        QString shell_cmd;
        if ("%[1]s" == "s" || "%[1]s" == "d") {
            shell_cmd = QString("%[3]spython3 -u -c %%1").arg(quotedPyCode);
        } else {
            shell_cmd = QString("%[3]spython3 -u -c %%1%%2").arg(quotedPyCode, FALLBACK_CMD);
        }
        `, mode, funcName, sudoPrefix))
	} else if sourceType == "javascript" {
		cpp.WriteString(buildJavaScriptShellCmdCPP(mode, funcName, sudoPrefix, "targetScript", false))
	} else if sourceType == "powershell" {
		cpp.WriteString(buildPowerShellShellCmdCPP(mode, funcName, sudoPrefix, "targetScript", false, action.Fields))
	} else {
		cpp.WriteString(fmt.Sprintf(`
        QString shell_cmd;
        if ("%[1]s" == "s" || "%[1]s" == "d") {
            shell_cmd = QString("source \"%%1\"; %[2]s%%2").arg(targetScript, escaped_args);
        } else {
            shell_cmd = QString("source \"%%1\"; %[2]s%%2%%3").arg(targetScript, escaped_args, FALLBACK_CMD);
        }
		`, mode, funcName))
	}

	cpp.WriteString(`
        QProcess *proc = new QProcess(this);
        QProcessEnvironment env = QProcessEnvironment::systemEnvironment();
        env.remove("QT_QPA_PLATFORMTHEME");
        env.remove("QT_STYLE_OVERRIDE");
        proc->setProcessEnvironment(env);
    `)

	if mode == "s" {
		cpp.WriteString(fmt.Sprintf(`
        execBtn_%[2]s->setEnabled(false);
        execBtn_%[2]s->setText("%[3]s");
        stopBtn_%[2]s->setEnabled(true);

        configureShellProcess(proc, shell_cmd, %[5]s);

        connect(stopBtn_%[2]s, &QPushButton::clicked, proc, [proc]() {
            proc->terminate();
            QTimer::singleShot(2000, proc, &QProcess::kill);
        });

        connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, closeOnFinish, this, execBtn_%[2]s, stopBtn_%[2]s](int exitCode, QProcess::ExitStatus status) {
            execBtn_%[2]s->setEnabled(true);
            execBtn_%[2]s->setText("%[4]s");
            stopBtn_%[2]s->setEnabled(false);
            if (status == QProcess::NormalExit && exitCode == 0) {
%[1]s
            }
            proc->deleteLater();
            malloc_trim(0);

            if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
        });

        proc->start();
			`, compStr, actIdxStr, cppStringLiteral(inProgressLabel), cppStringLiteral(execLabel), elevated))
	} else if mode == "d" {
		// Parity Restored: Detached mode correctly ported
		cpp.WriteString(fmt.Sprintf(`
		configureShellProcess(proc, shell_cmd, %[2]s);
        proc->startDetached();
        proc->deleteLater();
		%[1]s
        if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
			`, compStr, elevated))
	} else if mode == "g" || mode == "gc" {
		cpp.WriteString(fmt.Sprintf(`
        execBtn_%[2]s->setEnabled(false);
        execBtn_%[2]s->setText("%[3]s");
        stopBtn_%[2]s->setEnabled(true);

        configureShellProcess(proc, shell_cmd, %[5]s);
        proc->setProcessChannelMode(QProcess::MergedChannels);

        logOutputWindow_%[2]s->clear();
        if (!logToggle_%[2]s->isChecked()) {
            logToggle_%[2]s->setChecked(true);
        }

        connect(stopBtn_%[2]s, &QPushButton::clicked, proc, [proc]() {
            proc->terminate();
            QTimer::singleShot(2000, proc, &QProcess::kill);
        });

        connect(proc, &QProcess::readyReadStandardOutput, [proc, logOutputWindow_%[2]s]() {
            QString out = QString::fromUtf8(proc->readAllStandardOutput());
            logOutputWindow_%[2]s->append(stripAnsiEscapeCodes(out));
        });

        connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, closeOnFinish, this, execBtn_%[2]s, stopBtn_%[2]s, logOutputWindow_%[2]s](int exitCode, QProcess::ExitStatus status) {
            execBtn_%[2]s->setEnabled(true);
            execBtn_%[2]s->setText("%[4]s");
            stopBtn_%[2]s->setEnabled(false);
%[1]s
            proc->deleteLater();
            malloc_trim(0);

            if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
        });

        proc->start();
			`, compStr, actIdxStr, cppStringLiteral(inProgressLabel), cppStringLiteral(execLabel), elevated))
	} else {
		cpp.WriteString(fmt.Sprintf(`
        if (!configureTerminalProcess(proc, shell_cmd, %[2]s)) {
            proc->deleteLater();
            QMessageBox::warning(this, "Terminal unavailable", "No supported terminal emulator was found.");
            return;
        }

        connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
%[1]s
            proc->deleteLater();
            malloc_trim(0);

            if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
        });

        proc->start();
			`, compStr, elevated))
	}

	cpp.WriteString("\n    };\n")

	cpp.WriteString(fmt.Sprintf(`
    connect(execBtn_%[1]s, &QPushButton::clicked, [%[2]s, execLogic_%[1]s, this]() {
        QStringList args;
        for (const auto &p : %[2]s) {
            QWidget* widget = std::get<0>(p);
            QString fieldType = std::get<1>(p);
            QString dataType = std::get<2>(p);
            QString rawVal;
            if (!widget->isEnabled()) {
                if (fieldType == "CK") {
                    rawVal = "0";
                } else {
                    rawVal = "";
                }
            } else if (fieldType == "MCB") {
                CheckableListWidget* mcb = static_cast<CheckableListWidget*>(widget);
                if (!mcb->isValid()) {
                    QString fName = mcb->property("fieldName").toString();
                    QString maxStr = mcb->maxChecked > 0 ? QString::number(mcb->maxChecked) : "unlimited";
                    QMessageBox::warning(this, "Validation Error",
                        QString("The field '%%1' requires between %%2 and %%3 selections.")
                        .arg(fName).arg(mcb->minChecked).arg(maxStr));
                    return; // Abort execution immediately
                }

                if (mcb->maxChecked == 1) {
                    QStringList checked = mcb->checkedItems();
                    rawVal = checked.isEmpty() ? "" : checked.first();
                } else {
                    QStringList escaped;
                    for (const QString& item : mcb->checkedItems()) {
                        QString esc = item;
                        esc.replace("\"", "\"\"");
                        escaped.append("\"" + esc + "\"");
                    }
                    rawVal = escaped.join(",");
                }
            } else if (fieldType == "CB") {
                QComboBox* cb = static_cast<QComboBox*>(widget);
                QVariant data = cb->currentData();
                if (data.isValid() && !data.toString().isEmpty()) {
                    rawVal = data.toString();
                } else {
                    rawVal = cb->currentText();
                }
            } else if (fieldType == "CK") {
                rawVal = static_cast<QCheckBox*>(widget)->isChecked() ? "1" : "0";
            } else if (fieldType == "NUM") {
                if (QDoubleSpinBox *dsb = qobject_cast<QDoubleSpinBox*>(widget)) {
                    rawVal = QString::number(dsb->value());
                } else if (QSpinBox *sb = qobject_cast<QSpinBox*>(widget)) {
                    rawVal = QString::number(sb->value());
                }
            } else if (fieldType == "DATE") {
                rawVal = static_cast<QDateEdit*>(widget)->date().toString("yyyy-MM-dd");
            } else if (fieldType == "TIME") {
                rawVal = static_cast<QTimeEdit*>(widget)->time().toString("HH:mm:ss");
            } else if (fieldType == "LTXT") {
                rawVal = static_cast<QTextEdit*>(widget)->toPlainText();
            } else {
                rawVal = static_cast<QLineEdit*>(widget)->text();
            }

            if (QStringLiteral("%[3]s") == "python") {
                args.append(formatPythonArg(rawVal, dataType));
            } else if (QStringLiteral("%[3]s") == "javascript") {
                args.append(formatJsonArg(rawVal, dataType));
            } else if (QStringLiteral("%[3]s") == "powershell") {
                args.append(formatPowerShellArg(rawVal, dataType));
            } else {
                args.append(rawVal);
            }
        }
%[4]s
        execLogic_%[1]s(args, false);
    });
`, actIdxStr, fieldsArrName, sourceType, buildJavaScriptObjectWrapCPP(action, sourceType)))

	return cpp.String()
}
