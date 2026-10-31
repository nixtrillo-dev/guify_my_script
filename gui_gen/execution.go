// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"fmt"
	"sort"
	"strings"
)

// orderedParameterFields returns fields in script argument order while leaving
// the original field slice untouched for GUI construction.
func orderedParameterFields(fields []UIField) []UIField {
	indices := parameterOrderIndices(fields)
	ordered := make([]UIField, 0, len(fields))
	for _, index := range indices {
		ordered = append(ordered, fields[index])
	}
	return ordered
}

func parameterOrderIndices(fields []UIField) []int {
	indices := make([]int, len(fields))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := fields[indices[i]].ParameterPosition, fields[indices[j]].ParameterPosition
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return *left < *right
	})
	return indices
}

func parameterOrderCPP(fields []UIField) string {
	if len(fields) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("        {\n            QStringList orderedArgs;\n            const int parameterOrder[] = {")
	for i, index := range parameterOrderIndices(fields) {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%d", index)
	}
	b.WriteString("};\n            for (int index : parameterOrder) if (index < args.size()) orderedArgs.append(args.at(index));\n            args = orderedArgs;\n        }\n")
	return b.String()
}

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

	mode := action.GetMode()
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
	elevated := fmt.Sprintf("%t", action.Sudo)

	if sourceType == "go" || sourceType == "binary" || sourceType == "rust" {
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
	} else if sourceType == "powershell" {
		cpp.WriteString(buildPowerShellShellCmdCPP(mode, funcName, sudoPrefix, "SOURCE_SCRIPT", true, orderedParameterFields(action.Fields)))
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

	cpp.WriteString(buildWindowsShellCommandCPP(sourceType, funcName, "SOURCE_SCRIPT", mode))

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
            connect(btnStop, &QPushButton::clicked, this, [proc]() {
#ifdef Q_OS_WIN
                QProcess::execute("taskkill", QStringList() << "/PID" << QString::number(proc->processId()) << "/T" << "/F");
#else
                // A terminal launcher may exit after forwarding the request
                // to an existing GNOME Console instance, while the action is
                // still running. Stop the command by its script/function
                // pattern even when that launcher process has already exited.
                QString killPattern = QString("%%1.*%[1]s").arg(QFileInfo(SOURCE_SCRIPT).fileName());
                QProcess::execute("pkexec", QStringList() << "pkill" << "-9" << "-f" << killPattern);
#endif
                if (proc->state() != QProcess::NotRunning) proc->kill();
            });
`, funcName))
		} else {
			cpp.WriteString(fmt.Sprintf(`
            connect(btnStop, &QPushButton::clicked, this, [proc]() {
#ifdef Q_OS_WIN
                QProcess::execute("taskkill", QStringList() << "/PID" << QString::number(proc->processId()) << "/T" << "/F");
#else
                QString killPattern = QString("%%1.*%[1]s").arg(QFileInfo(SOURCE_SCRIPT).fileName());
                QProcess::execute("pkill", QStringList() << "-9" << "-f" << killPattern);
#endif
                if (proc->state() != QProcess::NotRunning) proc->kill();
            });
`, funcName))
		}

		if mode != "gc" {
			cpp.WriteString(fmt.Sprintf(`
            connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), this, [btnExec, btnStop](int, QProcess::ExitStatus) {
                btnExec->setEnabled(true);
                btnExec->setText("%[1]s"); // <-- Revert label to default Execution state
                btnStop->setEnabled(false);
            });
`, cppStringLiteral(execLabel)))
		}
		cpp.WriteString("        }\n")
	}

	if mode == "g" || mode == "gc" {
		// INJECT %[3]s for compStr in both success blocks
		cpp.WriteString(fmt.Sprintf(`
        logOutputWindow_%[1]s->clear();
        logOutputWindow_%[1]s->append("<i>[*] Instantiating execution thread...</i><br>");

        if ("%[2]s" == "gc") {
#ifdef Q_OS_WIN
            const QString commandLogPath = createCommandOutputLogPath();
            if (commandLogPath.isEmpty()) {
                proc->deleteLater();
                logOutputWindow_%[1]s->append("<font color='#a34747'>Could not create the Windows command output log.</font>");
                return;
            }
            const QString consoleInputSetup = QStringLiteral("Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public static class GuifyConsoleInput { [DllImport(\"kernel32.dll\", SetLastError=true, CharSet=CharSet.Unicode)] public static extern IntPtr CreateFile(string name, int access, uint share, IntPtr security, uint creation, uint flags, IntPtr template); [DllImport(\"kernel32.dll\", SetLastError=true)] public static extern bool SetStdHandle(int id, IntPtr handle); [DllImport(\"kernel32.dll\", SetLastError=true)] public static extern bool SetHandleInformation(IntPtr handle, uint mask, uint flags); }'; $consoleInput = [GuifyConsoleInput]::CreateFile('CONIN$', -1073741824, 3, [IntPtr]::Zero, 3, 0, [IntPtr]::Zero); if ($consoleInput.ToInt64() -eq -1 -or -not [GuifyConsoleInput]::SetHandleInformation($consoleInput, 1, 1) -or -not [GuifyConsoleInput]::SetStdHandle(-10, $consoleInput)) { throw 'Could not connect action input to the Windows console' }; ");
            QStringList encodedActionArgs;
            for (const auto &arg : actionArgs) encodedActionArgs << QString::fromLatin1(arg.toUtf8().toBase64());
            const QString encodedArgs = encodedActionArgs.join(",");
            const QString runnerType = QStringLiteral(R"CS(using System;
using System.Diagnostics;
using System.IO;
using System.Text;
using System.Threading.Tasks;
public static class GuifyActionRunner {
    static string Quote(string value) {
        char slash = (char)92, quote = (char)34;
        if (value.Length > 0 && value.IndexOf(' ') < 0 && value.IndexOf((char)9) < 0 && value.IndexOf((char)10) < 0 && value.IndexOf((char)13) < 0 && value.IndexOf(quote) < 0) return value;
        var result = new StringBuilder(); result.Append(quote); int slashes = 0;
        foreach (char c in value) {
            if (c == slash) { slashes++; continue; }
            if (c == quote) { result.Append(slash, slashes * 2 + 1); result.Append(quote); slashes = 0; continue; }
            result.Append(slash, slashes); slashes = 0; result.Append(c);
        }
        result.Append(slash, slashes * 2); result.Append(quote); return result.ToString();
    }
    public static int Run(string fileName, string encodedArgs, int argCount, string logPath) {
        var decoded = new string[argCount]; var parts = encodedArgs.Split(',');
        for (int i = 0; i < argCount; i++) decoded[i] = Encoding.UTF8.GetString(Convert.FromBase64String(parts[i]));
        var process = new Process(); var start = new ProcessStartInfo(fileName);
        start.Arguments = String.Join(" ", Array.ConvertAll(decoded, Quote)); start.UseShellExecute = false;
        start.CreateNoWindow = false; start.RedirectStandardOutput = true; start.RedirectStandardError = true;
        start.RedirectStandardInput = false; process.StartInfo = start;
        using (var log = new StreamWriter(new FileStream(logPath, FileMode.Append, FileAccess.Write, FileShare.ReadWrite), new UTF8Encoding(false))) {
            object gate = new object();
            Action<StreamReader, TextWriter> pump = delegate(StreamReader reader, TextWriter console) {
                char[] buffer = new char[256]; int count;
                while ((count = reader.Read(buffer, 0, buffer.Length)) > 0) {
                    lock (gate) { console.Write(buffer, 0, count); console.Flush(); log.Write(buffer, 0, count); log.Flush(); }
                }
            };
            process.Start();
            Task stdout = Task.Factory.StartNew(() => pump(process.StandardOutput, Console.Out));
            Task stderr = Task.Factory.StartNew(() => pump(process.StandardError, Console.Error));
            process.WaitForExit(); Task.WaitAll(stdout, stderr); return process.ExitCode;
        }
    }
})CS");
            const QString consoleCommand = consoleInputSetup + QStringLiteral("Add-Type -TypeDefinition ") + powershellQuote(runnerType) + QStringLiteral("; $actionExitCode = [GuifyActionRunner]::Run(") + powershellQuote(actionProgram) + QStringLiteral(", ") + powershellQuote(encodedArgs) + QStringLiteral(", ") + QString::number(encodedActionArgs.size()) + QStringLiteral(", ") + powershellQuote(commandLogPath) + QStringLiteral("); exit $actionExitCode");
            if (!configureTerminalProcess(proc, consoleCommand, false, false, true)) {
                QFile::remove(commandLogPath);
                proc->deleteLater();
                logOutputWindow_%[1]s->append("<font color='#a34747'>Could not open the Windows command console.</font>");
                return;
            }
            QFile *commandLog = new QFile(commandLogPath, this);
            QSharedPointer<qint64> commandLogOffset(new qint64(0));
            auto drainCommandLog = [commandLog, commandLogOffset, logOutputWindow_%[1]s]() {
                if (!commandLog->open(QIODevice::ReadOnly)) return;
                if (!commandLog->seek(*commandLogOffset)) {
                    commandLog->close();
                    return;
                }
                const QByteArray data = commandLog->readAll();
                *commandLogOffset = commandLog->pos();
                commandLog->close();
                if (data.isEmpty()) return;
                QTextCursor cursor = logOutputWindow_%[1]s->textCursor();
                cursor.movePosition(QTextCursor::End);
                logOutputWindow_%[1]s->setTextCursor(cursor);
                logOutputWindow_%[1]s->insertPlainText(stripAnsiEscapeCodes(QString::fromUtf8(data)));
                logOutputWindow_%[1]s->ensureCursorVisible();
            };
            QTimer *commandLogTimer = new QTimer(this);
            connect(commandLogTimer, &QTimer::timeout, this, drainCommandLog);
            commandLogTimer->start(100);
            connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
                    [proc, commandLogPath, commandLogTimer, drainCommandLog, logOutputWindow_%[1]s, btnExec, btnStop, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
                drainCommandLog();
                commandLogTimer->stop();
                if (status == QProcess::NormalExit && exitCode == 0) {
                    logOutputWindow_%[1]s->insertHtml("<br><font color='#10a37f'><b>[✔] Session successfully wrapped up.</b></font>");
%[3]s
                } else {
                    logOutputWindow_%[1]s->insertHtml(QString("<br><font color='#a34747'><b>[✖] Command exited with code %%1.</b></font>").arg(exitCode));
                }
                if (btnExec) { btnExec->setEnabled(true); btnExec->setText("%[6]s"); }
                if (btnStop) btnStop->setEnabled(false);
                QFile::remove(commandLogPath);
                commandLogTimer->deleteLater();
                proc->deleteLater();
                releaseMemory();
                if (closeOnFinish) QTimer::singleShot(100, this, &MainWindow::cleanupAndExit);
            });
            connect(proc, &QProcess::errorOccurred, this, [proc, commandLogPath, commandLogTimer, drainCommandLog, logOutputWindow_%[1]s, btnExec, btnStop](QProcess::ProcessError error) {
                if (error != QProcess::FailedToStart) return;
                commandLogTimer->stop();
                drainCommandLog();
                if (btnExec) { btnExec->setEnabled(true); btnExec->setText("%[6]s"); }
                if (btnStop) btnStop->setEnabled(false);
                logOutputWindow_%[1]s->append(QString("<br><font color='#a34747'><b>[✖] Could not start the action process: %%1</b></font>").arg(proc->errorString()));
                QFile::remove(commandLogPath);
                commandLogTimer->deleteLater();
                proc->deleteLater();
            });
            proc->start();
#else
            const QString fifoPath = QString("/tmp/panel_fifo_%%1_%%2").arg(APP_UUID).arg(rand() %% 100000);
            if (mkfifo(fifoPath.toUtf8().constData(), 0666) != 0) {
                proc->deleteLater();
                logOutputWindow_%[1]s->append("<font color='#a34747'>Could not create execution output stream.</font>");
                return;
            }

            // All supported GUI_COMMANDLINE terminals run this same FIFO
            // pipeline. The terminal child and QProcess stay alive until the
            // command exits, while cat streams tee output into the GUI.
            QString commandForTerminal = shell_cmd;
            if (%[4]s) {
                // Keep tee and the FIFO reader in the GUI user's context, as
                // in the original Konsole path. Only the requested action is
                // elevated; this avoids root-owned output-stream behavior.
                commandForTerminal = QString("pkexec bash -c %%1").arg(shellQuote(shell_cmd));
            }
            const QString terminalCommand = QString("set -o pipefail; { %%1; } 2>&1 | tee %%2")
                .arg(commandForTerminal, shellQuote(fifoPath));
            if (!configureTerminalProcess(proc, terminalCommand, false)) {
                QFile::remove(fifoPath);
                proc->deleteLater();
                logOutputWindow_%[1]s->append("<font color='#a34747'>No supported tracked terminal emulator was found.</font>");
                return;
            }

            QProcess *pipeReader = new QProcess(this);
            pipeReader->setProgram("cat");
            pipeReader->setArguments(QStringList() << fifoPath);
            connect(pipeReader, &QProcess::readyReadStandardOutput, [pipeReader, logOutputWindow_%[1]s]() {
                QString data = QString::fromUtf8(pipeReader->readAllStandardOutput());
                QTextCursor outputCursor = logOutputWindow_%[1]s->textCursor();
                outputCursor.movePosition(QTextCursor::End);
                logOutputWindow_%[1]s->setTextCursor(outputCursor);
                logOutputWindow_%[1]s->insertPlainText(stripAnsiEscapeCodes(data));
                logOutputWindow_%[1]s->ensureCursorVisible();
            });
            connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
                    [proc, pipeReader, fifoPath, logOutputWindow_%[1]s, btnExec, btnStop,
                     closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
                // Let cat receive EOF from tee and drain its last buffered
                // output before deleting the FIFO reader.
                if (!pipeReader->waitForFinished(1000)) {
                    pipeReader->terminate();
                    pipeReader->waitForFinished(500);
                }
                const QString remainingOutput = QString::fromUtf8(pipeReader->readAllStandardOutput());
                if (!remainingOutput.isEmpty()) {
                    QTextCursor outputCursor = logOutputWindow_%[1]s->textCursor();
                    outputCursor.movePosition(QTextCursor::End);
                    logOutputWindow_%[1]s->setTextCursor(outputCursor);
                    logOutputWindow_%[1]s->insertPlainText(stripAnsiEscapeCodes(remainingOutput));
                    logOutputWindow_%[1]s->ensureCursorVisible();
                }
                pipeReader->deleteLater();
                QFile::remove(fifoPath);

                QTextCursor outputCursor = logOutputWindow_%[1]s->textCursor();
                outputCursor.movePosition(QTextCursor::End);
                logOutputWindow_%[1]s->setTextCursor(outputCursor);
                if (status == QProcess::NormalExit && exitCode == 0) {
                    logOutputWindow_%[1]s->insertHtml("<br><font color='#10a37f'><b>[✔] Session successfully wrapped up.</b></font>");
                } else {
                    logOutputWindow_%[1]s->insertHtml(QString("<br><font color='#a34747'><b>[✖] Command exited with code %%1.</b></font>").arg(exitCode));
                }
                if (btnExec) {
                    btnExec->setEnabled(true);
                    btnExec->setText("%[6]s");
                }
                if (btnStop) btnStop->setEnabled(false);
%[3]s
                proc->deleteLater();
                releaseMemory();
                if (closeOnFinish) QTimer::singleShot(100, this, &MainWindow::cleanupAndExit);
            });
            connect(proc, &QProcess::errorOccurred, this,
                    [proc, pipeReader, fifoPath, logOutputWindow_%[1]s, btnExec, btnStop](QProcess::ProcessError error) {
                if (error != QProcess::FailedToStart) return;
                pipeReader->kill();
                pipeReader->deleteLater();
                QFile::remove(fifoPath);
                if (btnExec) {
                    btnExec->setEnabled(true);
                    btnExec->setText("%[6]s");
                }
                if (btnStop) btnStop->setEnabled(false);
                logOutputWindow_%[1]s->append("<br><font color='#a34747'><b>[✖] Could not start the terminal emulator.</b></font>");
                proc->deleteLater();
            });

            pipeReader->start();
            proc->start();
#endif
        } else {
            configureActionProcess(proc, shell_cmd, %[4]s, QStringLiteral("%[8]s"), SOURCE_SCRIPT, QStringLiteral("%[9]s"), args);
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
                releaseMemory();

                if (closeOnFinish) {
                    QTimer::singleShot(100, this, &MainWindow::cleanupAndExit);
                }
            });

            connect(proc, &QProcess::errorOccurred, this, [proc, logOutputWindow_%[1]s, btnExec, btnStop](QProcess::ProcessError error) {
                if (error != QProcess::FailedToStart) return;
                if (btnExec) { btnExec->setEnabled(true); btnExec->setText("%[6]s"); }
                if (btnStop) btnStop->setEnabled(false);
                logOutputWindow_%[1]s->append(QString("<br><font color='#a34747'><b>[✖] Could not start the action process: %%1</b></font>").arg(proc->errorString()));
                proc->deleteLater();
            });

            proc->start();
        }
			`, actIdxStr, mode, compStr, elevated, actIdxStr, cppStringLiteral(execLabel), "", cppStringLiteral(sourceType), cppStringLiteral(funcName)))
	} else if mode == "s" {
		cpp.WriteString(fmt.Sprintf(`
        configureActionProcess(proc, shell_cmd, %[2]s, QStringLiteral("%[3]s"), SOURCE_SCRIPT, QStringLiteral("%[4]s"), args);

        connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
            if (status == QProcess::NormalExit && exitCode == 0) {
%[1]s
            }
            proc->deleteLater();
            releaseMemory();

            if (closeOnFinish) {
                QTimer::singleShot(100, this, &MainWindow::cleanupAndExit);
            }
        });

        proc->start();
            `, compStr, elevated, cppStringLiteral(sourceType), cppStringLiteral(funcName)))
	} else if mode == "d" {
		cpp.WriteString(fmt.Sprintf(`
        configureActionProcess(proc, shell_cmd, %[1]s, QStringLiteral("%[3]s"), SOURCE_SCRIPT, QStringLiteral("%[4]s"), args);
        proc->startDetached();
%[2]s
        if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
            `, elevated, compStr, cppStringLiteral(sourceType), cppStringLiteral(funcName)))
	} else {
		cpp.WriteString(fmt.Sprintf(`
        if (!configureTerminalProcess(proc, shell_cmd, %[2]s, true)) {
            proc->deleteLater();
            QMessageBox::warning(this, "Terminal unavailable", "No supported terminal emulator was found.");
            return;
        }

#ifdef Q_OS_WIN
        if (!proc->startDetached()) {
            proc->deleteLater();
            QMessageBox::warning(this, "Terminal unavailable", "Could not start the Windows command console.");
            return;
        }
        proc->deleteLater();
#else
        connect(proc, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished), [proc, closeOnFinish, this](int exitCode, QProcess::ExitStatus status) {
%[1]s
            proc->deleteLater();
            releaseMemory();

            if (closeOnFinish) { QTimer::singleShot(100, this, &MainWindow::cleanupAndExit); }
        });

        proc->start();
#endif
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
                    if (!rawVal.isEmpty() && (dataType.toLower().trimmed() == "list" || dataType.toLower().trimmed() == "array")) {
                        rawVal = "[" + jsonEscapeString(rawVal) + "]";
                    }
                } else {
                    rawVal = joinCsvValues(mcb->checkedItems());
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
%[5]s
%[4]s
        execLogic_%[1]s(args, false);
    });
		`, actIdxStr, fieldsArrName, sourceType, buildJavaScriptObjectWrapCPP(UIAction{InvokeAsObject: action.InvokeAsObject, Fields: orderedParameterFields(action.Fields)}, sourceType), parameterOrderCPP(action.Fields)))

	return cpp.String()
}

// buildWindowsShellCommandCPP replaces the Bash command assembled by the
// existing Linux path with a PowerShell-compatible invocation. The Linux
// command construction remains unchanged under the platform guard.
func buildWindowsShellCommandCPP(sourceType, funcName, scriptExpr, mode string) string {
	var b strings.Builder
	b.WriteString("\n#ifdef Q_OS_WIN\n")
	switch sourceType {
	case "go", "rust", "binary":
		fmt.Fprintf(&b, "        QString actionProgram = %s;\n        QStringList actionArgs; actionArgs << QStringLiteral(\"%s\"); actionArgs.append(args);\n", scriptExpr, cppStringLiteral(funcName))
		fmt.Fprintf(&b, "        QStringList winArgs; winArgs << powershellQuote(%s) << powershellQuote(QStringLiteral(\"%s\"));\n", scriptExpr, cppStringLiteral(funcName))
		b.WriteString("        for (const auto &a : args) winArgs << powershellQuote(a);\n        shell_cmd = QStringLiteral(\"& \") + winArgs.join(\" \");\n")
	case "python":
		b.WriteString("        QString actionProgram = QStandardPaths::findExecutable(QStringLiteral(\"python\")); if (actionProgram.isEmpty()) actionProgram = QStringLiteral(\"python\");\n        QStringList actionArgs; actionArgs << QStringLiteral(\"-u\") << QStringLiteral(\"-c\") << pyCode;\n")
		b.WriteString("        shell_cmd = QStringLiteral(\"& python -u -c \") + powershellQuote(pyCode);\n")
	case "javascript":
		fmt.Fprintf(&b, "        QString actionProgram = QStringLiteral(\"node\"); QStringList actionArgs; actionArgs << QStringLiteral(\"--input-type=module\") << QStringLiteral(\"-e\") << jsRunner << %s << QStringLiteral(\"%s\"); actionArgs.append(args);\n", scriptExpr, cppStringLiteral(funcName))
		fmt.Fprintf(&b, "        QStringList winArgs; winArgs << powershellQuote(%s) << powershellQuote(QStringLiteral(\"%s\"));\n", scriptExpr, cppStringLiteral(funcName))
		b.WriteString("        for (const auto &a : args) winArgs << powershellQuote(a);\n        shell_cmd = QStringLiteral(\"& node --input-type=module -e \") + powershellQuote(jsRunner) + QStringLiteral(\" \") + winArgs.join(\" \");\n")
	case "powershell":
		b.WriteString("        QString actionProgram = QStringLiteral(\"powershell.exe\"); QStringList actionArgs; actionArgs << QStringLiteral(\"-NoLogo\") << QStringLiteral(\"-NoProfile\") << QStringLiteral(\"-ExecutionPolicy\") << QStringLiteral(\"Bypass\") << QStringLiteral(\"-Command\") << (psRunner + psArgv);\n")
		b.WriteString("        shell_cmd = QStringLiteral(\"& powershell.exe -NoProfile -ExecutionPolicy Bypass -Command \") + powershellQuote(psRunner) + psArgv;\n")
	default:
		b.WriteString("        QString actionProgram = QStringLiteral(\"powershell.exe\"); QStringList actionArgs; actionArgs << QStringLiteral(\"-NoProfile\") << QStringLiteral(\"-Command\") << shell_cmd;\n")
		b.WriteString("        shell_cmd = QStringLiteral(\"& bash -lc \") + powershellQuote(shell_cmd);\n")
	}
	b.WriteString("#endif\n")
	return b.String()
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
	cleanMode := NormalizeExecutionMode(mode)
	cond := fmt.Sprintf("\"%s\" == \"s\" || \"%s\" == \"d\"", cleanMode, cleanMode)
	if includeGuiModes {
		cond = fmt.Sprintf("\"%s\" == \"g\" || \"%s\" == \"s\" || \"%s\" == \"d\"", cleanMode, cleanMode, cleanMode)
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

const powershellRunner = `& {
$decodedArgs = @($args | ForEach-Object { [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($_)) })
$scriptPath = $decodedArgs[0]
$functionName = $decodedArgs[1]
Import-Module -Name $scriptPath -Force
$callArgs = @{}
if ($decodedArgs.Count -gt 2) {
    for ($i = 2; $i -lt $decodedArgs.Count; $i += 2) {
        $parameterName = $decodedArgs[$i].TrimStart('-')
        $callArgs[$parameterName] = $decodedArgs[$i + 1] | ConvertFrom-Json
    }
}
& $functionName @callArgs
}`

func buildPowerShellShellCmdCPP(mode, funcName, sudoPrefix, scriptExpr string, includeGuiModes bool, fields []UIField) string {
	cleanMode := NormalizeExecutionMode(mode)
	cond := fmt.Sprintf("\"%s\" == \"s\" || \"%s\" == \"d\"", cleanMode, cleanMode)
	if includeGuiModes {
		cond = fmt.Sprintf("\"%s\" == \"g\" || \"%s\" == \"s\" || \"%s\" == \"d\"", cleanMode, cleanMode, cleanMode)
	}
	var names strings.Builder
	for _, field := range fields {
		name := field.ArgName
		if name == "" {
			name = field.Name
		}
		names.WriteString(fmt.Sprintf(" << QStringLiteral(\"%s\")", cppStringLiteral(name)))
	}
	return fmt.Sprintf(`
        QString psRunner = QString::fromUtf8("%s");
        QString quotedPsRunner = shellQuote(psRunner);
        QString psArgv = QString(" ") + shellQuote(QString::fromLatin1(%s.toUtf8().toBase64())) + " " + shellQuote(QString::fromLatin1(QStringLiteral("%s").toUtf8().toBase64()));
        const QStringList psParamNames = QStringList()%s;
        for (int i = 0; i < args.size(); ++i) {
            if (i < psParamNames.size()) psArgv += " " + shellQuote(QString::fromLatin1(("-" + psParamNames.at(i)).toUtf8().toBase64()));
            const auto &a = args.at(i);
            psArgv += " " + shellQuote(QString::fromLatin1(a.toUtf8().toBase64()));
        }
        QString shell_cmd;
        if (%s) {
            shell_cmd = QString("%spwsh -NoProfile -NonInteractive -Command %%1%%2").arg(quotedPsRunner, psArgv);
        } else {
            shell_cmd = QString("%spwsh -NoProfile -NonInteractive -Command %%1%%2%%3").arg(quotedPsRunner, psArgv, FALLBACK_CMD);
        }
        `, cppStringLiteral(powershellRunner), scriptExpr, cppStringLiteral(funcName), names.String(), cond, sudoPrefix, sudoPrefix)
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
