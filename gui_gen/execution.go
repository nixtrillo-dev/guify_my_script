// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"fmt"
	"strings"
)

// buildExecutionLambda generates the Qt C++ lambda for executing the bash/binary processes,
// handling IO pipes, and connecting the execution button.
func buildExecutionLambda(action UIAction, label, actIdxStr, fieldsArrName, sourceType string) string {
	var cpp strings.Builder

	execLabel := action.ExecLabel
	if execLabel == "" {
		execLabel = fmt.Sprintf("Execute %s", label)
	}
	inProgressLabel := action.ExecInProgressLabel
	if inProgressLabel == "" {
		inProgressLabel = execLabel // Default to standard exec label if not overridden
	}

	// Build the completion refresh string based on JSON arrays
	var refreshCompletionStr strings.Builder
	if len(action.RefreshFieldsOnCompletion) > 0 {
		for _, fName := range action.RefreshFieldsOnCompletion {
			refreshCompletionStr.WriteString(fmt.Sprintf("                    refreshField(\"%s\", \"%s\");\n", cppStringLiteral(label), cppStringLiteral(fName)))
		}
	}
	compStr := refreshCompletionStr.String()

	mode := action.Mode
	if mode == "" {
		mode = "c"
	}
	funcName := action.Func
	isAttached := mode == "g" || mode == "gc" || mode == "s"

	execCapture := "this"
	if mode == "g" || mode == "gc" {
		execCapture += ", logOutputWindow_" + actIdxStr
	}

	cpp.WriteString(fmt.Sprintf(`
    auto execLogic_%[1]s = [%[2]s](const QStringList& args, bool closeOnFinish = false) {
        QString escaped_args;
        for (const auto &a : args) {
            escaped_args += " \"" + escapeArg(a) + "\"";
        }
        `, actIdxStr, execCapture))

	sudoPrefix := ""
	if action.Sudo {
		sudoPrefix = "pkexec "
	}

	if sourceType == "go" || sourceType == "binary" {
		cpp.WriteString(fmt.Sprintf(`
        QString shell_cmd;
        if ("%[1]s" == "g" || "%[1]s" == "s" || "%[1]s" == "d") {
            shell_cmd = QString("%[3]s\"%%1\" %[2]s%%2").arg(SOURCE_SCRIPT, escaped_args);
        } else {
            shell_cmd = QString("%[3]s\"%%1\" %[2]s%%2%%3").arg(SOURCE_SCRIPT, escaped_args, FALLBACK_CMD);
        }
        `, mode, funcName, sudoPrefix))
	} else {
		if action.Sudo {
			cpp.WriteString(fmt.Sprintf(`
        QString shell_cmd;
        if ("%[1]s" == "g" || "%[1]s" == "s" || "%[1]s" == "d") {
            shell_cmd = QString("pkexec bash -c 'source \"%%1\"; %[2]s%%2'").arg(SOURCE_SCRIPT, escaped_args);
        } else {
            shell_cmd = QString("pkexec bash -c 'source \"%%1\"; %[2]s%%2%%3'").arg(SOURCE_SCRIPT, escaped_args, FALLBACK_CMD);
        }
        `, mode, funcName))
		} else {
			cpp.WriteString(fmt.Sprintf(`
        QString shell_cmd;
        if ("%[1]s" == "g" || "%[1]s" == "s" || "%[1]s" == "d") {
            shell_cmd = QString("source \"%%1\"; %[2]s%%2").arg(SOURCE_SCRIPT, escaped_args);
        } else {
            shell_cmd = QString("source \"%%1\"; %[2]s%%2%%3").arg(SOURCE_SCRIPT, escaped_args, FALLBACK_CMD);
        }
        `, mode, funcName))
		}
	}

	cpp.WriteString(`
        QProcess *proc = new QProcess(this);
        QProcessEnvironment env = QProcessEnvironment::systemEnvironment();
        env.remove("QT_QPA_PLATFORMTHEME");
        env.remove("QT_STYLE_OVERRIDE");
        proc->setProcessEnvironment(env);
        `)

	if isAttached {
		cpp.WriteString(fmt.Sprintf(`
        QPushButton *btnExec = this->findChild<QPushButton*>("execBtn_%[1]s");
        QPushButton *btnStop = this->findChild<QPushButton*>("stopBtn_%[1]s");
        QPushButton *logToggle = this->findChild<QPushButton*>("logToggle_%[1]s");

        // Auto-expand the logs when execution starts
        if (logToggle && !logToggle->isChecked()) {
            logToggle->setChecked(true);
        }

        if (btnExec && btnStop) {
            btnExec->setEnabled(false);
            btnExec->setText("%[2]s"); // <-- Change label to In-Progress state
            btnStop->setEnabled(true);
            btnStop->disconnect();
`, actIdxStr, cppStringLiteral(inProgressLabel)))

		if action.Sudo {
			cpp.WriteString(fmt.Sprintf(`
            connect(btnStop, &QPushButton::clicked, proc, [proc]() {
                if (proc->state() != QProcess::NotRunning) {
                    QString killCmd = QString("pkexec pkill -9 -f \"%%1.*%[1]s\"").arg(QFileInfo(SOURCE_SCRIPT).fileName());
                    QProcess::execute("bash", QStringList() << "-c" << killCmd);
                    proc->kill();
                }
            });
`, funcName))
		} else {
			cpp.WriteString(`
            connect(btnStop, &QPushButton::clicked, proc, [proc]() {
                if (proc->state() != QProcess::NotRunning) {
                    proc->kill();
                }
            });
`)
		}

		cpp.WriteString(fmt.Sprintf(`
            connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), this, [btnExec, btnStop](int, QProcess::ExitStatus) {
                btnExec->setEnabled(true);
                btnExec->setText("%[1]s"); // <-- Revert label to default Execution state
                btnStop->setEnabled(false);
            });
        }
`, cppStringLiteral(execLabel)))
	}

	if mode == "g" || mode == "gc" {
		// INJECT %[3]s for compStr in both success blocks
		cpp.WriteString(fmt.Sprintf(`
        logOutputWindow_%[1]s->clear();
        logOutputWindow_%[1]s->append("<i>[*] Instantiating execution thread...</i><br>");

        if ("%[2]s" == "gc") {
            QString fifoPath = QString("/tmp/panel_fifo_%%1_%%2").arg(APP_UUID).arg(rand() %% 100000);
            mkfifo(fifoPath.toUtf8().constData(), 0666);

            QProcess *pipeReader = new QProcess(this);
            pipeReader->setProgram("cat");
            pipeReader->setArguments(QStringList() << fifoPath);

            connect(pipeReader, &QProcess::readyReadStandardOutput, [pipeReader, logOutputWindow_%[1]s]() {
                QString data = QString::fromUtf8(pipeReader->readAllStandardOutput());
                QTextCursor cursor = logOutputWindow_%[1]s->textCursor();
                cursor.movePosition(QTextCursor::End);
                logOutputWindow_%[1]s->setTextCursor(cursor);
                logOutputWindow_%[1]s->insertPlainText(stripAnsiEscapeCodes(data));
                logOutputWindow_%[1]s->ensureCursorVisible();
            });

            proc->setProgram("konsole");
            proc->setArguments(QStringList() << "-e" << "bash" << "-c" << QString("{ %%1; } 2>&1 | tee %%2").arg(shell_cmd, fifoPath));

            connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, pipeReader, fifoPath, logOutputWindow_%[1]s, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
                pipeReader->terminate();
                pipeReader->waitForFinished(500);
                pipeReader->deleteLater();
                unlink(fifoPath.toUtf8().constData());

                QTextCursor cursor = logOutputWindow_%[1]s->textCursor();
                cursor.movePosition(QTextCursor::End);
                logOutputWindow_%[1]s->setTextCursor(cursor);
                logOutputWindow_%[1]s->insertHtml("<br><font color='#10a37f'><b>[✔] Session successfully wrapped up.</b></font>");
%[3]s
                proc->deleteLater();
                malloc_trim(0);

                if (closeOnFinish) {
                    QTimer::singleShot(100, this, &MainWindow::cleanupAndExit);
                }
            });

            pipeReader->start();
            proc->start();

        } else {
            proc->setProgram("bash");
            proc->setArguments(QStringList() << "-c" << shell_cmd);
            proc->setProcessChannelMode(QProcess::MergedChannels);

            connect(proc, &QProcess::readyReadStandardOutput, [proc, logOutputWindow_%[1]s]() {
                QString newOutput = QString::fromUtf8(proc->readAllStandardOutput());
                QTextCursor cursor = logOutputWindow_%[1]s->textCursor();
                cursor.movePosition(QTextCursor::End);
                logOutputWindow_%[1]s->setTextCursor(cursor);
                logOutputWindow_%[1]s->insertPlainText(stripAnsiEscapeCodes(newOutput));
                logOutputWindow_%[1]s->ensureCursorVisible();
            });

            connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, logOutputWindow_%[1]s, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
                QTextCursor cursor = logOutputWindow_%[1]s->textCursor();
                cursor.movePosition(QTextCursor::End);
                logOutputWindow_%[1]s->setTextCursor(cursor);

                if (status == QProcess::NormalExit && exitCode == 0) {
                    logOutputWindow_%[1]s->insertHtml("<br><font color='#10a37f'><b>[✔] Process completed successfully.</b></font>");
%[3]s
                } else {
                    logOutputWindow_%[1]s->insertHtml(QString("<br><font color='#a34747'><b>[✖] Process crashed or exited with error code: %%1</b></font>").arg(exitCode));
                }
                logOutputWindow_%[1]s->ensureCursorVisible();
                proc->deleteLater();
                malloc_trim(0);

                if (closeOnFinish) {
                    QTimer::singleShot(100, this, &MainWindow::cleanupAndExit);
                }
            });

            proc->start();
        }
            `, actIdxStr, mode, compStr))
	} else if mode == "s" {
		cpp.WriteString(fmt.Sprintf(`
        proc->setProgram("bash");
        proc->setArguments(QStringList() << "-c" << shell_cmd);

        connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
            if (status == QProcess::NormalExit && exitCode == 0) {
%[1]s
            }
            proc->deleteLater();
            malloc_trim(0);

            if (closeOnFinish) {
                QTimer::singleShot(100, this, &MainWindow::cleanupAndExit);
            }
        });

        proc->start();
            `, compStr))
	} else if mode == "d" {
		cpp.WriteString(fmt.Sprintf(`
        proc->setProgram("bash");
        proc->setArguments(QStringList() << "-c" << (shell_cmd + " &"));
        proc->startDetached();
%[1]s
        if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
            `, compStr))
	} else {
		cpp.WriteString(fmt.Sprintf(`
        proc->setProgram("konsole");
        proc->setArguments(QStringList() << "-e" << "bash" << "-c" << shell_cmd);

        connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
%[1]s
            proc->deleteLater();
            malloc_trim(0);

            if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
        });

        proc->start();
            `, compStr))
	}

	cpp.WriteString("\n    };\n")

	cpp.WriteString(fmt.Sprintf(`
    connect(execBtn_%[1]s, &QPushButton::clicked, [%[2]s, execLogic_%[1]s, this]() {
        QStringList args;
        for (const auto &p : %[2]s) {
            if (!p.first->isEnabled()) {
                if (p.second == "CK") {
                    args.append("0");
                } else {
                    args.append("");
                }
            } else if (p.second == "MCB") {
                CheckableListWidget* mcb = static_cast<CheckableListWidget*>(p.first);
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
                    args.append(checked.isEmpty() ? "" : checked.first());
                } else {
                    QStringList escaped;
                    for (const QString& item : mcb->checkedItems()) {
                        QString esc = item;
                        esc.replace("\"", "\"\"");
                        escaped.append("\"" + esc + "\"");
                    }
                    args.append(escaped.join(","));
                }
            } else if (p.second == "CB") {
                QComboBox* cb = static_cast<QComboBox*>(p.first);
                QVariant data = cb->currentData();
                if (data.isValid() && !data.toString().isEmpty()) {
                    args.append(data.toString());
                } else {
                    args.append(cb->currentText());
                }
           } else if (p.second == "CK") {
                args.append(static_cast<QCheckBox*>(p.first)->isChecked() ? "1" : "0");
            } else if (p.second == "NUM") {
                if (QDoubleSpinBox *dsb = qobject_cast<QDoubleSpinBox*>(p.first)) {
                    args.append(QString::number(dsb->value()));
                } else if (QSpinBox *sb = qobject_cast<QSpinBox*>(p.first)) {
                    args.append(QString::number(sb->value()));
                }
            } else if (p.second == "DATE") {
                args.append(static_cast<QDateEdit*>(p.first)->date().toString("yyyy-MM-dd"));
            } else if (p.second == "TIME") {
                args.append(static_cast<QTimeEdit*>(p.first)->time().toString("HH:mm:ss"));
            } else if (p.second == "LTXT") {
                args.append(static_cast<QTextEdit*>(p.first)->toPlainText());
            } else {
                args.append(static_cast<QLineEdit*>(p.first)->text());
            }
        }
        execLogic_%[1]s(args, false);
    });
        `, actIdxStr, fieldsArrName))

	return cpp.String()
}
