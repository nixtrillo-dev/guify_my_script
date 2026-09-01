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
	} else if sourceType == "python" {
		cpp.WriteString(fmt.Sprintf(`
        QString py_args = args.join(", ");
        QString modName = QFileInfo(SOURCE_SCRIPT).completeBaseName();
        QString pyCode = QString("import sys, os; sys.path.insert(0, os.path.dirname(%%1)); import %%2; %%2.%[2]s(%%3)").arg(escapePythonString(SOURCE_SCRIPT), modName, py_args);
        QString quotedPyCode = shellQuote(pyCode);

        QString shell_cmd;
        if ("%[1]s" == "g" || "%[1]s" == "s" || "%[1]s" == "d") {
            shell_cmd = QString("%[3]spython3 -u -c %%1").arg(quotedPyCode);
        } else {
            shell_cmd = QString("%[3]spython3 -u -c %%1%%2").arg(quotedPyCode, FALLBACK_CMD);
        }
        `, mode, funcName, sudoPrefix))
	} else if sourceType == "javascript" {
		cpp.WriteString(buildJavaScriptShellCmdCPP(mode, funcName, sudoPrefix, "SOURCE_SCRIPT", true))
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

const javascriptNodeRunner = `import { pathToFileURL } from 'node:url';
import { readFileSync } from 'node:fs';
const script = process.argv[1];
const funcName = process.argv[2];
if (typeof funcName !== 'string' || !/^[A-Za-z_][A-Za-z0-9_]*$/.test(funcName)) {
  throw new Error('Invalid function name');
}
const callArgs = process.argv.slice(3).map((s) => JSON.parse(s));
let fn;
try {
  const mod = await import(pathToFileURL(script).href);
  fn = mod[funcName];
} catch {
  fn = undefined;
}
if (typeof fn !== 'function') {
  const src = readFileSync(script, 'utf8');
  const shim = src + '\nexport { ' + funcName + ' };\n';
  const mod = await import('data:text/javascript;charset=utf-8,' + encodeURIComponent(shim));
  fn = mod[funcName];
}
if (typeof fn !== 'function') {
  throw new Error('Function is not exported and could not be loaded: ' + funcName);
}
const result = await fn(...callArgs);
if (result !== undefined) {
  if (typeof result === 'string') console.log(result);
  else console.log(JSON.stringify(result));
}
`

func buildJavaScriptShellCmdCPP(mode, funcName, sudoPrefix, scriptExpr string, includeGuiModes bool) string {
	cond := fmt.Sprintf("\"%s\" == \"s\" || \"%s\" == \"d\"", mode, mode)
	if includeGuiModes {
		cond = fmt.Sprintf("\"%s\" == \"g\" || \"%s\" == \"s\" || \"%s\" == \"d\"", mode, mode, mode)
	}
	return fmt.Sprintf(`
        QString jsRunner = QString::fromUtf8("%s");
        QString quotedJsRunner = shellQuote(jsRunner);
        QString jsArgv = QString(" ") + shellQuote(%s) + " " + shellQuote(QStringLiteral("%s"));
        for (const auto &a : args) {
            jsArgv += " " + shellQuote(a);
        }
        QString shell_cmd;
        if (%s) {
            shell_cmd = QString("%snode --input-type=module -e %%1%%2").arg(quotedJsRunner, jsArgv);
        } else {
            shell_cmd = QString("%snode --input-type=module -e %%1%%2%%3").arg(quotedJsRunner, jsArgv, FALLBACK_CMD);
        }
        `, cppStringLiteral(javascriptNodeRunner), scriptExpr, cppStringLiteral(funcName), cond, sudoPrefix, sudoPrefix)
}

func buildJavaScriptObjectWrapCPP(action UIAction, sourceType string) string {
	if sourceType != "javascript" || !action.InvokeAsObject || len(action.Fields) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("        {\n            QStringList jsParts;\n            const QStringList jsKeys = QStringList()")
	for _, f := range action.Fields {
		key := f.ArgName
		if key == "" {
			key = f.Name
		}
		b.WriteString(fmt.Sprintf(" << QStringLiteral(\"%s\")", cppStringLiteral(key)))
	}
	b.WriteString(";\n")
	b.WriteString(`            for (int i = 0; i < args.size() && i < jsKeys.size(); ++i) {
                jsParts.append(QString("\"%1\":%2").arg(jsKeys.at(i), args.at(i)));
            }
            args = QStringList() << ("{" + jsParts.join(",") + "}");
        }
`)
	return b.String()
}

