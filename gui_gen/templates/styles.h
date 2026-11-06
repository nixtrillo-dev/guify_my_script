// Style helper functions
static QString appLineEditStyle() {
    return isLightTheme()
        ? "QLineEdit { background-color: #ffffff; color: #202124; border: 1px solid #c9d1d9; padding: 4px; border-radius: 4px; selection-background-color: #10a37f; selection-color: #ffffff; } QLineEdit:focus { border: 1px solid #10a37f; }"
        : "QLineEdit { background-color: #23272a; color: #ffffff; border: 1px solid #3a3f44; padding: 4px; border-radius: 4px; selection-background-color: #10a37f; } QLineEdit:focus { border: 1px solid #10a37f; }";
}

static QString appSubtleLabelStyle(const QString &extra = QString()) {
    return QString("color: %1; %2").arg(themeVal("#b0b5b9", "#5f6b76"), extra);
}

static QString appStrongLabelStyle(const QString &extra = QString()) {
    return QString("color: %1; font-weight: bold; background: transparent; %2").arg(themeVal("#ffffff", "#202124"), extra);
}

static QString appSeparatorStyle() {
    return QString("border: none; background-color: %1; max-height: 1px; margin-top: 5px; margin-bottom: 5px;").arg(themeVal("#43494e", "#d0d7de"));
}

static QString appLogToggleStyle() {
    return QString("QPushButton { text-align: left; background: transparent; border: none; font-weight: bold; color: %1; padding: 4px 0px; } QPushButton:hover, QPushButton:focus { color: %2; }")
        .arg(themeVal("#b0b5b9", "#5f6b76"), themeVal("#ffffff", "#202124"));
}

static QString appLogOutputStyle() {
    QString fixedFamily = QFontDatabase::systemFont(QFontDatabase::FixedFont).family();
    QString fontFamilies = QString("'%1', 'DejaVu Sans Mono', 'Liberation Mono', 'Noto Sans Mono', 'Consolas', 'Courier New', monospace").arg(fixedFamily);

    return isLightTheme()
        ? QString("font-family: %1; background-color: #f6f8fa; color: #1f2328; border: 1px solid #d0d7de; font-size: 11px;").arg(fontFamilies)
        : QString("font-family: %1; background-color: #161819; color: #a9b1d6; border: 1px solid #3a3f44; font-size: 11px;").arg(fontFamilies);
}

// A QTextEdit owns a QTextDocument, whose default font is not reliably
// replaced by a QSS font-family rule alone. Set the QFont on both the widget
// and document, in addition to the QSS font-family, so spaces retain the fixed
// width needed for terminal-style, column-aligned output.
static void applyLogOutputStyle(QTextEdit *textEdit) {
    QFont logFont = QFontDatabase::systemFont(QFontDatabase::FixedFont);
    logFont.setStyleHint(QFont::Monospace);
    logFont.setFixedPitch(true);
    logFont.setPointSize(11);

    textEdit->setFont(logFont);
    textEdit->document()->setDefaultFont(logFont);
    textEdit->setCurrentFont(logFont);
    textEdit->setStyleSheet(appLogOutputStyle());
}

static QString appAccentButtonStyle(const QString &accent, const QString &hoverAccent = QString()) {
    Q_UNUSED(hoverAccent);
    return QString("QPushButton { background-color: %1; border: 1px solid %2; color: %2; padding: 8px; border-radius: 4px; font-weight: bold; } QPushButton:hover, QPushButton:focus { background-color: %2; color: #ffffff; } QPushButton:disabled { border: 1px solid %3; color: %4; background-color: %5; }")
        .arg(themeVal("#23272a", "#ffffff"), accent, themeVal("#43494e", "#d0d7de"), themeVal("#7f8c8d", "#8c959f"), themeVal("#1b1e20", "#f6f8fa"));
}

static QString appSimpleAccentButtonStyle(const QString &accent, const QString &hoverAccent = QString()) {
    const QString hover = hoverAccent.isEmpty() ? accent : hoverAccent;
    return QString("QPushButton { background-color: %1; border: 1px solid %2; color: %2; font-weight: bold; padding: 10px; border-radius: 4px; } QPushButton:hover, QPushButton:focus { background-color: %3; color: #ffffff; }")
        .arg(themeVal("#23272a", "#ffffff"), accent, hover);
}

static QString appSecondaryButtonStyle() {
    return QString("QPushButton { background-color: %1; border: 1px solid %2; color: %2; font-weight: bold; padding: 10px; border-radius: 4px; } QPushButton:hover, QPushButton:focus { background-color: %2; color: %3; }")
        .arg(themeVal("#23272a", "#ffffff"), themeVal("#b0b5b9", "#5f6b76"), themeVal("#1b1e20", "#ffffff"));
}

static QString appComboViewportStyle() {
    return QString("background-color: %1; color: %2; border: none;").arg(themeVal("#23272a", "#ffffff"), themeVal("#ffffff", "#202124"));
}

static QString appComboPopupStyle() {
    return QString(
        "QFrame { background-color: %1; border: 1px solid %2; }"
        "QComboBox { background-color: %1; color: %3; border: 1px solid %2; padding: 5px 25px 5px 5px; border-radius: 4px; combobox-popup: 1; }"
        "QComboBox:hover, QComboBox:focus { border: 1px solid #10a37f; }"
        "QComboBox::drop-down { subcontrol-origin: padding; subcontrol-position: top right; width: 24px; border-left: 1px solid %2; border-top-right-radius: 3px; border-bottom-right-radius: 3px; background-color: %4; }"
        "QComboBox::drop-down:hover { background-color: %2; }"
        "QComboBox QAbstractItemView { background-color: %1; color: %3; border: 1px solid %2; selection-background-color: %5; selection-color: %8; outline: none; }"
        "QComboBox QAbstractItemView::item:selected, QComboBox QAbstractItemView::item:hover { background-color: %5; color: %8; }"
        "QListView { background-color: %1; color: %3; border: 1px solid %2; selection-background-color: %5; selection-color: %8; outline: none; }"
        "QListView::item:selected, QListView::item:hover { background-color: %5; color: %8; }"
        "QAbstractScrollArea::corner { background: %6; border: none; }"
        "QScrollBar:vertical { background: %6; width: 10px; margin: 0px; border: none; }"
        "QScrollBar::handle:vertical { background: %2; min-height: 20px; border-radius: 3px; }"
        "QScrollBar::handle:vertical:hover { background: #10a37f; }"
        "QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical { height: 0px; background: %6; border: none; }"
        "QScrollBar::up-arrow:vertical, QScrollBar::down-arrow:vertical { background: transparent; border: none; width: 0px; height: 0px; }"
        "QScrollBar::add-page:vertical, QScrollBar::sub-page:vertical { background: %7; border: none; }"
    ).arg(themeVal("#23272a", "#ffffff"), themeVal("#3a3f44", "#c9d1d9"), themeVal("#ffffff", "#202124"), themeVal("#2a2e32", "#f6f8fa"), themeVal("#3a3f44", "#cfe3ff"), themeVal("#1b1e20", "#f6f8fa"), themeVal("#2b2b2b", "#eaeef2"), themeVal("#ffffff", "#202124"));
}

static void drawAppComboArrow(QPainter *painter, const QRect &bounds, bool up, const QColor &color) {
    if (!painter || bounds.isEmpty()) return;
    const int width = 7;
    const int height = 4;
    const int left = bounds.center().x() - width / 2;
    const int top = bounds.center().y() - height / 2;
    QPolygon triangle;
    if (up) {
        triangle << QPoint(left, top + height) << QPoint(left + width, top + height)
                 << QPoint(left + width / 2, top);
    } else {
        triangle << QPoint(left, top) << QPoint(left + width, top)
                 << QPoint(left + width / 2, top + height);
    }
    painter->save();
    painter->setPen(Qt::NoPen);
    painter->setBrush(color);
    painter->drawPolygon(triangle);
    painter->restore();
}

class AppComboArrowOverlay final : public QWidget {
public:
    explicit AppComboArrowOverlay(QComboBox *comboBox) : QWidget(comboBox), m_comboBox(comboBox) {
        setObjectName(QStringLiteral("AppComboArrowOverlay"));
        setAttribute(Qt::WA_TransparentForMouseEvents);
        setAttribute(Qt::WA_NoSystemBackground);
        setAutoFillBackground(false);
        if (m_comboBox) {
            m_comboBox->installEventFilter(this);
            m_view = m_comboBox->view();
            if (m_view) {
                m_view->installEventFilter(this);
            }
            m_popupOpen = m_view && m_view->isVisible();
            updateGeometryFromCombo();
        }
    }

protected:
    bool eventFilter(QObject *watched, QEvent *event) override {
        if (m_comboBox && watched == m_comboBox.data() &&
            (event->type() == QEvent::Resize || event->type() == QEvent::Show ||
             event->type() == QEvent::StyleChange)) {
            updateGeometryFromCombo();
        } else if (m_comboBox && watched == m_comboBox.data() &&
                   (event->type() == QEvent::Enter || event->type() == QEvent::Leave ||
                    event->type() == QEvent::FocusIn || event->type() == QEvent::FocusOut)) {
            update();
        } else if (m_view && watched == m_view.data()) {
            if (event->type() == QEvent::Show || event->type() == QEvent::Hide) {
                m_popupOpen = event->type() == QEvent::Show;
                update();
            }
        }
        return QWidget::eventFilter(watched, event);
    }

    void paintEvent(QPaintEvent *) override {
        QPainter painter(this);
        const QColor color(m_comboBox && !m_comboBox->isEnabled()
            ? themeVal("#7f8c8d", "#8c959f")
            : themeVal("#b0b5b9", "#5f6b76"));
        drawAppComboArrow(&painter, rect(), m_popupOpen, color);
    }

private:
    void updateGeometryFromCombo() {
        if (!m_comboBox) return;
        setGeometry(m_comboBox->width() - 24, 0, 24, m_comboBox->height());
        raise();
        update();
    }

    QPointer<QComboBox> m_comboBox;
    QPointer<QAbstractItemView> m_view;
    bool m_popupOpen = false;
};

static void installAppComboArrowOverlay(QComboBox *comboBox) {
    if (!comboBox || comboBox->findChild<QWidget *>(QStringLiteral("AppComboArrowOverlay"))) return;
    new AppComboArrowOverlay(comboBox);
}

class AppComboBoxItemDelegate final : public QStyledItemDelegate {
public:
    using QStyledItemDelegate::QStyledItemDelegate;

    void paint(QPainter *painter, const QStyleOptionViewItem &option,
               const QModelIndex &index) const override {
        const bool highlighted = option.state.testFlag(QStyle::State_MouseOver) ||
                                 option.state.testFlag(QStyle::State_Selected);
        if (!highlighted) {
            QStyledItemDelegate::paint(painter, option, index);
            return;
        }

        painter->save();
        painter->fillRect(option.rect, isLightTheme() ? QColor("#cfe3ff") : QColor("#3a3f44"));
        painter->setFont(option.font);
        painter->setPen(isLightTheme() ? QColor("#202124") : QColor("#ffffff"));
        int contentLeft = option.rect.left() + 8;
        const QIcon icon = qvariant_cast<QIcon>(index.data(Qt::DecorationRole));
        if (!icon.isNull()) {
            const QSize iconSize = option.decorationSize.isValid() ? option.decorationSize : QSize(16, 16);
            const QRect iconRect(contentLeft, option.rect.center().y() - iconSize.height() / 2,
                                 iconSize.width(), iconSize.height());
            icon.paint(painter, iconRect);
            contentLeft = iconRect.right() + 6;
        }
        painter->drawText(QRect(contentLeft, option.rect.top(), option.rect.right() - contentLeft - 4,
                                option.rect.height()), Qt::AlignLeft | Qt::AlignVCenter,
                          index.data(Qt::DisplayRole).toString());
        painter->restore();
    }
};

static void applyItemViewSelectionStyle(QAbstractItemView *view) {
    if (!view) return;
    view->setMouseTracking(true);
    view->viewport()->setAttribute(Qt::WA_Hover, true);
    if (!dynamic_cast<AppComboBoxItemDelegate *>(view->itemDelegate())) {
        view->setItemDelegate(new AppComboBoxItemDelegate(view));
    }
}

static void applyComboBoxPopupStyle(QComboBox *comboBox) {
    if (!comboBox || !comboBox->view()) return;
    comboBox->view()->setStyleSheet(appComboPopupStyle());
    comboBox->view()->viewport()->setStyleSheet(appComboViewportStyle());
    applyItemViewSelectionStyle(comboBox->view());
    if (comboBox->view()->parentWidget()) {
        comboBox->view()->parentWidget()->setStyleSheet(appComboPopupStyle());
        comboBox->view()->parentWidget()->setAttribute(Qt::WA_StyledBackground, true);
    }
}

static void prepareAppFileDialog(QFileDialog *dialog) {
    if (!dialog) return;
#ifdef Q_OS_LINUX
    dialog->setOption(QFileDialog::DontUseNativeDialog, true);
    dialog->setSizeGripEnabled(false);
    dialog->setStyleSheet(QStringLiteral("QFileDialog { background-color: %1; color: %2; }")
        .arg(themeVal("#1b1e20", "#f6f8fa"), themeVal("#ffffff", "#202124")));
    for (QComboBox *combo : dialog->findChildren<QComboBox *>()) {
        combo->setMouseTracking(true);
        combo->setAttribute(Qt::WA_Hover, true);
        installAppComboArrowOverlay(combo);
        combo->setStyleSheet(QString(
            "QComboBox { background-color: %1; color: %2; border: 1px solid %3; padding: 4px 24px 4px 5px; }"
            "QComboBox:hover, QComboBox:focus { border: 1px solid #10a37f; }"
            "QComboBox::drop-down { subcontrol-origin: padding; subcontrol-position: top right; width: 22px; border-left: 1px solid %3; background-color: %4; }"
            "QComboBox::drop-down:hover { background-color: %3; }"
        ).arg(themeVal("#23272a", "#ffffff"), themeVal("#ffffff", "#202124"),
              themeVal("#3a3f44", "#c9d1d9"), themeVal("#2a2e32", "#f6f8fa")));
        applyComboBoxPopupStyle(combo);
    }
    for (QLineEdit *lineEdit : dialog->findChildren<QLineEdit *>()) {
        if (QCompleter *completer = lineEdit->completer()) {
            if (QAbstractItemView *popup = completer->popup()) {
                popup->setStyleSheet(appComboPopupStyle());
                popup->viewport()->setStyleSheet(appComboViewportStyle());
                applyItemViewSelectionStyle(popup);
            }
        }
    }
#endif
}

static QString chooseAppOpenFile(QWidget *parent, const QString &caption,
                                 const QString &directory, const QString &filter = QString()) {
    QFileDialog dialog(parent, caption, directory);
    prepareAppFileDialog(&dialog);
    dialog.setAcceptMode(QFileDialog::AcceptOpen);
    dialog.setFileMode(QFileDialog::ExistingFile);
    if (!filter.isEmpty()) dialog.setNameFilter(filter);
    if (dialog.exec() != QDialog::Accepted) return QString();
    const QStringList files = dialog.selectedFiles();
    return files.isEmpty() ? QString() : files.first();
}

static QStringList chooseAppOpenFiles(QWidget *parent, const QString &caption,
                                      const QString &directory, const QString &filter = QString()) {
    QFileDialog dialog(parent, caption, directory);
    prepareAppFileDialog(&dialog);
    dialog.setAcceptMode(QFileDialog::AcceptOpen);
    dialog.setFileMode(QFileDialog::ExistingFiles);
    if (!filter.isEmpty()) dialog.setNameFilter(filter);
    if (dialog.exec() != QDialog::Accepted) return {};
    return dialog.selectedFiles();
}

static QString chooseAppDirectory(QWidget *parent, const QString &caption,
                                  const QString &directory) {
    QFileDialog dialog(parent, caption, directory);
    prepareAppFileDialog(&dialog);
    dialog.setAcceptMode(QFileDialog::AcceptOpen);
    dialog.setFileMode(QFileDialog::Directory);
    dialog.setOption(QFileDialog::ShowDirsOnly, true);
    if (dialog.exec() != QDialog::Accepted) return QString();
    const QStringList directories = dialog.selectedFiles();
    return directories.isEmpty() ? QString() : directories.first();
}

static QString appMcbStyle() {
    return QString(
        "QListWidget { background-color: %1; color: %2; border: 1px solid %3; border-radius: 4px; outline: none; padding: 6px 4px 4px 4px; } "
        "QListWidget:focus { border: 1px solid #10a37f; } "
        "QListWidget::item { padding: 2px; border: 1px solid transparent; border-radius: 3px; } "
        "QListWidget::item:hover { background-color: %4; } "
        "QListWidget::item:focus, QListWidget::item:selected { background-color: rgba(16, 163, 127, 0.15); color: %7; border: 1px solid #10a37f; } "
        "QListWidget QScrollBar:vertical { background: %5; width: 10px; margin: 0px; border: none; } "
        "QListWidget QScrollBar::handle:vertical { background: %6; min-height: 20px; border-radius: 3px; } "
        "QListWidget QScrollBar::handle:vertical:hover { background: #10a37f; }"
    ).arg(themeVal("#23272a", "#ffffff"), themeVal("#ffffff", "#202124"), themeVal("#3a3f44", "#c9d1d9"), themeVal("#2c3135", "#f6f8fa"), themeVal("#1b1e20", "#f6f8fa"), themeVal("#43494e", "#c9d1d9"), themeVal("#ffffff", "#202124"));
}
