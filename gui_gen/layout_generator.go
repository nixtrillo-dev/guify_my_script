// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"fmt"
	"strconv"
	"strings"
)

// buildQuickLaunchRows generates the C++ code for the advanced menu's --trigger commands.
func buildQuickLaunchRows(actions []UIAction) string {
	var advancedRows strings.Builder
	quickLaunchCount := 0

	for actIdx, action := range actions {
		if !action.QuickLaunch {
			continue
		}

		label := cppStringLiteral(action.Label)
		advancedRows.WriteString(fmt.Sprintf(`
    QWidget *triggerRow_%[1]d = new QWidget();
    QVBoxLayout *triggerRowLayout_%[1]d = new QVBoxLayout(triggerRow_%[1]d);
    triggerRowLayout_%[1]d->setContentsMargins(0, 0, 0, 0);
    triggerRowLayout_%[1]d->setSpacing(4);

    QLabel *triggerLabel_%[1]d = new QLabel("%[2]s");
    triggerLabel_%[1]d->setObjectName("QuickTriggerLabel_%[1]d");
    triggerLabel_%[1]d->setStyleSheet(appStrongLabelStyle());
    triggerRowLayout_%[1]d->addWidget(triggerLabel_%[1]d);

    QWidget *triggerCommandFrame_%[1]d = new QWidget();
    QHBoxLayout *triggerCommandLayout_%[1]d = new QHBoxLayout(triggerCommandFrame_%[1]d);
    triggerCommandLayout_%[1]d->setContentsMargins(0, 0, 0, 0);
    triggerCommandLayout_%[1]d->setSpacing(8);

    QString triggerPath_%[1]d = QCoreApplication::applicationFilePath();
    QString triggerName_%[1]d = "%[2]s";
    triggerPath_%[1]d.replace("\"", "\\\"");
    triggerName_%[1]d.replace("\"", "\\\"");
    QString triggerCommand_%[1]d = QString("\"%%1\" --trigger \"%%2\"").arg(triggerPath_%[1]d, triggerName_%[1]d);

    QLineEdit *triggerPathEdit_%[1]d = new QLineEdit(triggerCommand_%[1]d);
    triggerPathEdit_%[1]d->setReadOnly(true);
    triggerPathEdit_%[1]d->setFocusPolicy(Qt::TabFocus);

    QPushButton *copyTriggerBtn_%[1]d = new QPushButton("Copy");
    copyTriggerBtn_%[1]d->setFixedWidth(80);
    copyTriggerBtn_%[1]d->setFocusPolicy(Qt::TabFocus);
    connect(copyTriggerBtn_%[1]d, &QPushButton::clicked, [triggerPathEdit_%[1]d]() {
        QGuiApplication::clipboard()->setText(triggerPathEdit_%[1]d->text());
    });

    triggerCommandLayout_%[1]d->addWidget(triggerPathEdit_%[1]d, 1);
    triggerCommandLayout_%[1]d->addWidget(copyTriggerBtn_%[1]d);
    triggerRowLayout_%[1]d->addWidget(triggerCommandFrame_%[1]d);
    triggerListLayout->addWidget(triggerRow_%[1]d);
`, actIdx, label))
		quickLaunchCount++
	}

	if quickLaunchCount == 0 {
		advancedRows.WriteString(`
    QLabel *noTriggersLabel = new QLabel("No quick launch actions are configured for --trigger commands.");
    noTriggersLabel->setObjectName("NoTriggersLabel");
    noTriggersLabel->setAlignment(Qt::AlignCenter);
    noTriggersLabel->setStyleSheet(QString("color: %1; background: transparent;").arg(themeVal("#7f8c8d", "#8c959f")));
    triggerListLayout->addWidget(noTriggersLabel);
`)
	}

	return advancedRows.String()
}

// buildActionTabs generates the C++ UI code for the main application tabs, fields, and action buttons.
func buildActionTabs(actions []UIAction, sourceType string, actionOnStartup string, closeAfterStartup bool, defaultTabOnStartup string) string {
	var cpp strings.Builder

	for actIdx, action := range actions {
		label := action.Label
		mode := action.Mode
		if mode == "" {
			mode = "c"
		}
		sanitizedLabel := sanitizeIdentifier(label)
		actIdxStr := strconv.Itoa(actIdx)

		cpp.WriteString(fmt.Sprintf("\n    // Tab Layout Setup: %s (Mode: %s)\n", label, mode))
		cpp.WriteString("    currentTab = new QWidget();\n")

		// 1. Outer layout for the tab (holds ScrollArea + Logs + Action Buttons)
		cpp.WriteString(fmt.Sprintf("    QVBoxLayout *tabOuterLayout_%s = new QVBoxLayout(currentTab);\n", actIdxStr))
		cpp.WriteString(fmt.Sprintf("    tabOuterLayout_%s->setSpacing(10);\n", actIdxStr))
		cpp.WriteString(fmt.Sprintf("    tabOuterLayout_%s->setContentsMargins(15, 15, 15, 15);\n", actIdxStr))

		// 2. The ScrollArea strictly for the input fields
		cpp.WriteString(fmt.Sprintf("    QScrollArea *fieldsScroll_%s = new QScrollArea();\n", actIdxStr))
		cpp.WriteString(fmt.Sprintf("    fieldsScroll_%s->setWidgetResizable(true);\n", actIdxStr))
		cpp.WriteString(fmt.Sprintf("    fieldsScroll_%s->setFrameShape(QFrame::NoFrame);\n", actIdxStr))
		cpp.WriteString(fmt.Sprintf("    fieldsScroll_%s->setStyleSheet(\"QScrollArea { background: transparent; } QWidget#scrollContent_%s { background: transparent; }\");\n", actIdxStr, actIdxStr))

		// 3. The inner scrolling content layout where the fields actually go
		cpp.WriteString(fmt.Sprintf("    QWidget *scrollContent_%s = new QWidget();\n", actIdxStr))
		cpp.WriteString(fmt.Sprintf("    scrollContent_%s->setObjectName(\"scrollContent_%s\");\n", actIdxStr, actIdxStr))
		cpp.WriteString(fmt.Sprintf("    currentLayout = new QVBoxLayout(scrollContent_%s);\n", actIdxStr))
		cpp.WriteString("    currentLayout->setSpacing(10);\n")
		cpp.WriteString("    currentLayout->setContentsMargins(0, 0, 10, 0);\n")

		cpp.WriteString(fmt.Sprintf("    fieldsScroll_%s->setWidget(scrollContent_%s);\n", actIdxStr, actIdxStr))
		cpp.WriteString(fmt.Sprintf("    tabOuterLayout_%s->addWidget(fieldsScroll_%s, 1);\n", actIdxStr, actIdxStr))

		fieldsArrName := fmt.Sprintf("fields_%s_%s", sanitizedLabel, actIdxStr)
		cpp.WriteString(fmt.Sprintf("        \n    std::vector<std::tuple<QWidget*, QString, QString>> %s;\n", fieldsArrName))

		hasAdvanced := false
		for _, field := range action.Fields {
			if field.AdvancedMenu {
				hasAdvanced = true
				break
			}
		}

		if hasAdvanced {
			cpp.WriteString(fmt.Sprintf(`
    QPushButton *advToggle_%[1]s = new QPushButton("▶ Advanced Settings");
    advToggle_%[1]s->setCheckable(true);
    advToggle_%[1]s->setStyleSheet("QPushButton { text-align: left; background: transparent; border: none; font-weight: bold; color: #10a37f; padding: 4px 0px; } QPushButton:hover, QPushButton:focus { color: #14c79b; }");
    advToggle_%[1]s->setFocusPolicy(Qt::TabFocus);

    QWidget *advFrame_%[1]s = new QWidget();
    QVBoxLayout *advLayout_%[1]s = new QVBoxLayout(advFrame_%[1]s);
    advLayout_%[1]s->setContentsMargins(10, 0, 0, 0);
    advLayout_%[1]s->setSpacing(10);
    advFrame_%[1]s->setVisible(false);

    connect(advToggle_%[1]s, &QPushButton::toggled, [advToggle_%[1]s, advFrame_%[1]s](bool checked) {
        advFrame_%[1]s->setVisible(checked);
        advToggle_%[1]s->setText(checked ? "▼ Advanced Settings" : "▶ Advanced Settings");
    });
`, actIdxStr))
		}

		for fIdx, field := range action.Fields {
			fIdxStr := strconv.Itoa(fIdx)
			varSuffix := fmt.Sprintf("%s_%s_%s", sanitizedLabel, actIdxStr, fIdxStr)

			targetLayout := "currentLayout"
			if field.AdvancedMenu {
				targetLayout = fmt.Sprintf("advLayout_%s", actIdxStr)
			}

			widgetCode := buildWidgetCPP(field, label, actIdxStr, varSuffix, targetLayout, fieldsArrName)
			cpp.WriteString(widgetCode)
		}

		if hasAdvanced {
			cpp.WriteString(fmt.Sprintf(`
    currentLayout->addWidget(advToggle_%[1]s);
    currentLayout->addWidget(advFrame_%[1]s);
`, actIdxStr))
		}

		// Pack fields to the top of the scroll area
		cpp.WriteString("\n    currentLayout->addStretch();\n")

		if mode == "g" || mode == "gc" {
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
    logOutputWindow_%[1]s->setStyleSheet(appLogOutputStyle());

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

		isAttached := mode == "g" || mode == "gc" || mode == "s"

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

		// Connect the final elements up to tabOuterLayout instead of currentLayout
		cpp.WriteString(fmt.Sprintf(`
    tabOuterLayout_%[1]s->addLayout(actionBtnLayout_%[1]s);
    m_tabs->addTab(currentTab, "%[2]s");
        `, actIdxStr, label))

		// Generate the execution lambda and button bindings
		cpp.WriteString(buildExecutionLambda(action, label, actIdxStr, fieldsArrName, sourceType))

		if action.QuickLaunch {
			cpp.WriteString(fmt.Sprintf(`
    auto triggerQuickLaunch_%[1]s = [this, execLogic_%[1]s]() {
        QStringList defArgs;
`, actIdxStr))
			for _, field := range action.Fields {
				defVal := getDefaultString(field.Default)
				if field.Type == "CK" && defVal != "1" {
					defVal = "0"
				}
				cpp.WriteString(fmt.Sprintf("        defArgs.append(getRuntimeDefault(\"%s\", \"%s\", \"%s\"));\n", label, field.Name, cppStringLiteral(defVal)))
			}
			cpp.WriteString(fmt.Sprintf(`
        execLogic_%[1]s(defArgs, false);
    };

    m_quickActions["%[2]s"] = triggerQuickLaunch_%[1]s;

    QAction *trayAct_%[1]s = new QAction("⚡ %[2]s", this);
    QPushButton *mainMenuBtn_%[1]s = new QPushButton("⚡ Quick Launch: %[2]s");
    mainMenuBtn_%[1]s->setObjectName("QuickLaunchBtn_%[1]s");
    mainMenuBtn_%[1]s->setStyleSheet(appAccentButtonStyle("#10a37f"));
    mainMenuBtn_%[1]s->setFocusPolicy(Qt::TabFocus);

    quickLaunchLayout->addWidget(mainMenuBtn_%[1]s);
    connect(trayAct_%[1]s, &QAction::triggered, triggerQuickLaunch_%[1]s);
    connect(mainMenuBtn_%[1]s, &QPushButton::clicked, triggerQuickLaunch_%[1]s);
    m_trayMenu->addAction(trayAct_%[1]s);
`, actIdxStr, label))
		}

		closeStartupDefStr := "0"
		if closeAfterStartup {
			closeStartupDefStr = "1"
		}

		cpp.WriteString(fmt.Sprintf(`
    if (!qApp->arguments().contains("--settings") && getRuntimeDefault("App Settings", "Action On Startup", "%[3]s") == "%[2]s") {
        bool closeOnFin = (getRuntimeDefault("App Settings", "Close After Startup Action", "%[4]s") == "1");

        QTimer::singleShot(250, this, [this, execLogic_%[1]s, closeOnFin]() {
            QStringList defArgs;
`, actIdxStr, label, actionOnStartup, closeStartupDefStr))

		for _, field := range action.Fields {
			defVal := getDefaultString(field.Default)
			if field.Type == "CK" && defVal != "1" {
				defVal = "0"
			}
			cpp.WriteString(fmt.Sprintf("            defArgs.append(getRuntimeDefault(\"%s\", \"%s\", \"%s\"));\n", label, field.Name, cppStringLiteral(defVal)))
		}

		cpp.WriteString(fmt.Sprintf(`
            execLogic_%[1]s(defArgs, closeOnFin);
        });
    }
`, actIdxStr))
	}

	// Tab Open Refresh Logic
	hasTabRefreshes := false
	var tabRefreshCPP strings.Builder
	tabRefreshCPP.WriteString("\n    connect(m_tabs, &QTabWidget::currentChanged, this, [this](int index) {\n")
	tabRefreshCPP.WriteString("        switch(index) {\n")

	for actIdx, action := range actions {
		if len(action.RefreshFieldsOnTabOpen) > 0 {
			hasTabRefreshes = true
			tabRefreshCPP.WriteString(fmt.Sprintf("            case %d:\n", actIdx+1)) // Offset by +1 for Main Menu Tab
			for _, fName := range action.RefreshFieldsOnTabOpen {
				tabRefreshCPP.WriteString(fmt.Sprintf("                refreshField(\"%s\", \"%s\");\n", cppStringLiteral(action.Label), cppStringLiteral(fName)))
			}
			tabRefreshCPP.WriteString("                break;\n")
		}
	}
	tabRefreshCPP.WriteString("        }\n")
	tabRefreshCPP.WriteString("    });\n")

	if hasTabRefreshes {
		cpp.WriteString(tabRefreshCPP.String())
	}

	if defaultTabOnStartup != "" {
		cpp.WriteString(fmt.Sprintf(`
    // Set default tab on startup
    for (int i = 0; i < m_tabs->count(); ++i) {
        if (m_tabs->tabText(i) == "%s") {
            m_tabs->setCurrentIndex(i);
            break;
        }
    }
`, cppStringLiteral(defaultTabOnStartup)))
	}

	return cpp.String()
}
