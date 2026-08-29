// AppComboBox implementation
void AppComboBox::showPopup() {
    QComboBox::showPopup();
    m_popupOpen = true;
    update();
}

void AppComboBox::hidePopup() {
    m_popupOpen = false;
    update();
    QComboBox::hidePopup();
}

void AppComboBox::paintEvent(QPaintEvent *event) {
    QComboBox::paintEvent(event);

    QPainter painter(this);
    painter.setPen(QColor(!isEnabled() ? themeVal("#7f8c8d", "#8c959f") : themeVal("#ffffff", "#202124")));
    painter.drawText(QRect(width() - 25, 1, 24, height() - 2), Qt::AlignCenter,
                     m_popupOpen ? QString::fromUtf8("▲") : QString::fromUtf8("▼"));
}

// AppDialog implementation
AppDialog::AppDialog(QWidget *parent) : QDialog(parent) {
    setWindowFlags(Qt::Dialog | Qt::FramelessWindowHint | Qt::WindowMinMaxButtonsHint);
    setAttribute(Qt::WA_StyledBackground, true);
    setMouseTracking(true);
    qApp->installEventFilter(this);

    QVBoxLayout *rootLayout = new QVBoxLayout(this);
    rootLayout->setContentsMargins(1, 1, 1, 1);
    rootLayout->setSpacing(0);

    QWidget *titleBar = new QWidget(this);
    titleBar->setObjectName("AppDialogTitleBar");
    titleBar->setFixedHeight(34);
    titleBar->installEventFilter(this);

    QHBoxLayout *titleLayout = new QHBoxLayout(titleBar);
    titleLayout->setContentsMargins(10, 3, 5, 3);
    titleLayout->setSpacing(6);

    m_titleLabel = new QLabel(titleBar);
    m_titleLabel->setObjectName("AppDialogTitleLabel");
    m_titleLabel->setAttribute(Qt::WA_TransparentForMouseEvents);
    titleLayout->addWidget(m_titleLabel);
    titleLayout->addStretch();

    QPushButton *minimizeButton = new QPushButton("—", titleBar);
    minimizeButton->setObjectName("AppDialogMinBtn");
    minimizeButton->setFixedSize(26, 24);
    connect(minimizeButton, &QPushButton::clicked, this, &QWidget::showMinimized);
    titleLayout->addWidget(minimizeButton);

    m_maxButton = new QPushButton("🗖", titleBar);
    m_maxButton->setObjectName("AppDialogMaxBtn");
    m_maxButton->setFixedSize(26, 24);
    connect(m_maxButton, &QPushButton::clicked, this, [this]() { toggleMaximized(); });
    titleLayout->addWidget(m_maxButton);

    QPushButton *closeButton = new QPushButton(QString::fromUtf8("✕"), titleBar);
    closeButton->setObjectName("AppDialogCloseBtn");
    closeButton->setFixedSize(26, 24);
    connect(closeButton, &QPushButton::clicked, this, &QDialog::reject);
    titleLayout->addWidget(closeButton);
    rootLayout->addWidget(titleBar);

    QWidget *content = new QWidget(this);
    content->setObjectName("AppDialogContent");
    m_contentLayout = new QVBoxLayout(content);
    rootLayout->addWidget(content, 1);
}

AppDialog::~AppDialog() {
    qApp->removeEventFilter(this);
}

QVBoxLayout *AppDialog::contentLayout() const {
    return m_contentLayout;
}

bool AppDialog::event(QEvent *event) {
    const bool handled = QDialog::event(event);
    if (event->type() == QEvent::WindowTitleChange && m_titleLabel) {
        m_titleLabel->setText(windowTitle());
    } else if (event->type() == QEvent::WindowStateChange && m_maxButton) {
        m_maxButton->setText(isMaximized() ? "🗗" : "🗖");
    }
    return handled;
}

bool AppDialog::eventFilter(QObject *watched, QEvent *event) {
    if (watched->objectName() == "AppDialogTitleBar") {
        if (event->type() == QEvent::ContextMenu) {
            QContextMenuEvent *contextEvent = static_cast<QContextMenuEvent*>(event);
            showWindowControlMenu(this, contextEvent->globalPos(), m_maxButton);
            event->accept();
            return true;
        }

        if (event->type() == QEvent::MouseButtonPress) {
            QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
            if (mouseEvent->button() == Qt::LeftButton && windowHandle()) {
                windowHandle()->startSystemMove();
                event->accept();
                return true;
            }
        }
    }

    if (event->type() == QEvent::MouseMove) {
        QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
        const Qt::Edges edges = resizeEdges(mapFromGlobal(mouseEvent->globalPosition().toPoint()));
        if (edges) setCursorForEdges(edges);
        else if (cursor().shape() != Qt::ArrowCursor) setCursor(Qt::ArrowCursor);
    } else if (event->type() == QEvent::MouseButtonPress) {
        QMouseEvent *mouseEvent = static_cast<QMouseEvent*>(event);
        const Qt::Edges edges = resizeEdges(mapFromGlobal(mouseEvent->globalPosition().toPoint()));
        if (mouseEvent->button() == Qt::LeftButton && edges && windowHandle()) {
            windowHandle()->startSystemResize(edges);
            event->accept();
            return true;
        }
    }
    return QDialog::eventFilter(watched, event);
}

Qt::Edges AppDialog::resizeEdges(const QPoint &pos) const {
    if (isMaximized()) return {};

    constexpr int border = 8;
    Qt::Edges edges = {};
    if (pos.x() <= border) edges |= Qt::LeftEdge;
    if (pos.x() >= width() - border) edges |= Qt::RightEdge;
    if (pos.y() <= border) edges |= Qt::TopEdge;
    if (pos.y() >= height() - border) edges |= Qt::BottomEdge;
    return edges;
}

void AppDialog::setCursorForEdges(Qt::Edges edges) {
    if ((edges & Qt::LeftEdge) && (edges & Qt::TopEdge)) setCursor(Qt::SizeBDiagCursor);
    else if ((edges & Qt::RightEdge) && (edges & Qt::BottomEdge)) setCursor(Qt::SizeBDiagCursor);
    else if ((edges & Qt::RightEdge) && (edges & Qt::TopEdge)) setCursor(Qt::SizeFDiagCursor);
    else if ((edges & Qt::LeftEdge) && (edges & Qt::BottomEdge)) setCursor(Qt::SizeFDiagCursor);
    else if (edges & Qt::LeftEdge || edges & Qt::RightEdge) setCursor(Qt::SizeHorCursor);
    else setCursor(Qt::SizeVerCursor);
}

void AppDialog::toggleMaximized() {
    if (isMaximized()) {
        showNormal();
        m_maxButton->setText("🗖");
    } else {
        showMaximized();
        m_maxButton->setText("🗗");
    }
}

// InternalListWidget implementation
void InternalListWidget::mouseReleaseEvent(QMouseEvent *event) {
    QListWidgetItem *itm = itemAt(event->pos());
    if (itm) {
        QRect rect = visualItemRect(itm);
        // 26px provides a safe boundary to separate the checkbox from the text.
        if (event->pos().x() > rect.x() + 26) {
            itm->setCheckState(itm->checkState() == Qt::Checked ? Qt::Unchecked : Qt::Checked);
        }
    }
    QListWidget::mouseReleaseEvent(event);
}

// CheckableListWidget implementation
CheckableListWidget::CheckableListWidget(QWidget *parent) : QWidget(parent) {
    QVBoxLayout *layout = new QVBoxLayout(this);
    layout->setContentsMargins(0, 0, 0, 0);
    layout->setSpacing(6);

    searchBox = new QLineEdit(this);
    searchBox->setObjectName("CheckableSearchBox");
    searchBox->setPlaceholderText("Search options...");
    searchBox->setClearButtonEnabled(true);
    searchBox->setStyleSheet(appLineEditStyle());
    searchBox->hide();

    headerWidget = new QWidget(this);
    QHBoxLayout *headerLayout = new QHBoxLayout(headerWidget);
    headerLayout->setContentsMargins(4, 0, 4, 0);
    headerLayout->setSpacing(10);

    statusLabel = new QLabel(this);
    statusLabel->setObjectName("CheckableStatusLabel");
    statusLabel->setStyleSheet(appSubtleLabelStyle("font-size: 11px; font-weight: bold;"));

    toggleAllBtn = new QPushButton(this);
    toggleAllBtn->setCursor(Qt::PointingHandCursor);
    toggleAllBtn->setStyleSheet(QString("QPushButton { background: transparent; border: none; color: #10a37f; text-decoration: underline; font-size: 11px; padding: 0; } QPushButton:hover { color: #14c79b; } QPushButton:disabled { color: %1; text-decoration: none; }").arg(themeVal("#5a6268", "#8c959f")));

    headerLayout->addWidget(statusLabel);
    headerLayout->addStretch();
    headerLayout->addWidget(toggleAllBtn);
    headerWidget->hide();

    listWidget = new InternalListWidget(this);

    // Define the visual separator
    separatorItem = new QListWidgetItem("────────── All Options ──────────");
    separatorItem->setFlags(Qt::NoItemFlags); // Disables selection/checking to act purely as a divider
    separatorItem->setTextAlignment(Qt::AlignCenter);
    listWidget->addItem(separatorItem);

    layout->addWidget(searchBox);
    layout->addWidget(headerWidget);
    layout->addWidget(listWidget);

    // Filter Logic
    connect(searchBox, &QLineEdit::textChanged, this, [this](const QString &text) {
        for (int i = 0; i < listWidget->count(); ++i) {
            QListWidgetItem *itm = listWidget->item(i);
            if (itm == separatorItem) continue;

            bool match = itm->text().contains(text, Qt::CaseInsensitive);
            itm->setHidden(!match);
        }

        // Let updateHeader handle the separator visibility entirely
        updateHeader();
    });

    // Toggle All Button Logic
    connect(toggleAllBtn, &QPushButton::clicked, this, [this]() {
        if (isUpdating) return;
        isUpdating = true;

        int sepRow = listWidget->row(separatorItem);
        int checkedCount = sepRow;
        bool doSelectAll = (maxChecked == 0 && checkedCount == 0);

        if (doSelectAll) {
            for (int i = sepRow + 1; i < listWidget->count(); ++i) {
                QListWidgetItem *itm = listWidget->item(i);
                if (!itm->isHidden() && itm->checkState() == Qt::Unchecked) {
                    itm->setCheckState(Qt::Checked);
                    QListWidgetItem *clone = new QListWidgetItem(itm->text());
                    clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
                    clone->setCheckState(Qt::Checked);
                    listWidget->insertItem(listWidget->row(separatorItem), clone);

                    i++; // Offset the index shift caused by the insertion
                }
            }
        } else { // Deselect All
            // Uncheck bottom section
            for (int i = sepRow + 1; i < listWidget->count(); ++i) {
                listWidget->item(i)->setCheckState(Qt::Unchecked);
            }
            // Purge top section
            for(int i = listWidget->row(separatorItem) - 1; i >= 0; --i) {
                delete listWidget->takeItem(i);
            }
        }

        updateHeader();
        isUpdating = false;
    });

    // Sync Logic
    connect(listWidget, &QListWidget::itemChanged, this, [this](QListWidgetItem *item) {
        if (isUpdating || item == separatorItem) return;
        isUpdating = true;

        int sepRow = listWidget->row(separatorItem);
        int itemRow = listWidget->row(item);
        bool isTopItem = (itemRow < sepRow);
        bool checked = (item->checkState() == Qt::Checked);

        // Limit Enforcement
        if (!isTopItem && checked && maxChecked > 0) {
            int checkedCount = sepRow;
            if (checkedCount >= maxChecked) {
                item->setCheckState(Qt::Unchecked);
                checked = false; // Revert locally
            }
        }

        QString text = item->text();

        if (isTopItem) {
            if (!checked) {
                // Turn off the corresponding item below the separator
                for (int i = listWidget->row(separatorItem) + 1; i < listWidget->count(); ++i) {
                    if (listWidget->item(i)->text() == text) {
                        listWidget->item(i)->setCheckState(Qt::Unchecked);
                        break;
                    }
                }
                delete listWidget->takeItem(itemRow); // Remove clone
            }
        } else {
            if (checked) {
                QListWidgetItem *clone = new QListWidgetItem(text);
                clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
                clone->setCheckState(Qt::Checked);
                listWidget->insertItem(listWidget->row(separatorItem), clone);
            } else {
                // Find and delete the clone above the separator
                for (int i = 0; i < listWidget->row(separatorItem); ++i) {
                    if (listWidget->item(i)->text() == text) {
                        delete listWidget->takeItem(i);
                        break;
                    }
                }
            }
        }

        updateHeader();
        isUpdating = false;
    });
}

void CheckableListWidget::updateHeader() {
    int sepRow = listWidget->row(separatorItem);
    int checkedCount = sepRow;
    int totalItems = listWidget->count() - 1 - checkedCount; // Total minus separator minus clones

    if (maxChecked > 0) {
        statusLabel->setText(QString("%1 of %2 selected").arg(checkedCount).arg(maxChecked));
    } else {
        statusLabel->setText(QString("%1 option(s) selected").arg(checkedCount));
    }

    bool hasSearch = !searchBox->text().isEmpty();

    if (hasSearch) {
        toggleAllBtn->hide();
    } else {
        toggleAllBtn->show();
        if (maxChecked > 0) {
            toggleAllBtn->setText("Deselect All");
            toggleAllBtn->setEnabled(checkedCount > 0);
        } else {
            if (checkedCount > 0) {
                toggleAllBtn->setText("Deselect All");
                toggleAllBtn->setEnabled(true);
            } else {
                toggleAllBtn->setText("Select All");
                toggleAllBtn->setEnabled(totalItems > 0);
            }
        }
    }

    headerWidget->setVisible(totalItems > 0 || checkedCount > 0);

    if (!hasSearch) {
        separatorItem->setHidden(checkedCount == 0); // Hide separator if nothing is selected
    } else {
        // Check if there are any visible clones (selected items) matching the search
        bool hasVisibleClones = false;
        for (int i = 0; i < sepRow; ++i) {
            if (!listWidget->item(i)->isHidden()) {
                hasVisibleClones = true;
                break;
            }
        }
        separatorItem->setHidden(!hasVisibleClones);
    }

    if (totalItems <= 7 && !hasSearch) {
        searchBox->hide();
    } else {
        searchBox->show();
    }
}

void CheckableListWidget::addItem(const QString &text, bool checked) {
    QListWidgetItem *item = new QListWidgetItem(text);
    item->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
    item->setCheckState(checked ? Qt::Checked : Qt::Unchecked);
    listWidget->addItem(item); // Native insert puts this at the bottom, below the separator

    if (checked) {
        QListWidgetItem *clone = new QListWidgetItem(text);
        clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
        clone->setCheckState(Qt::Checked);
        listWidget->insertItem(listWidget->row(separatorItem), clone);
    }
    updateHeader();
}

void CheckableListWidget::addItem(const QString &text, const QString &userData, bool checked) {
    QListWidgetItem *item = new QListWidgetItem(text, listWidget);
    item->setData(Qt::UserRole, userData); // Store your underlying alias value here
    item->setFlags(item->flags() | Qt::ItemIsUserCheckable);
    item->setCheckState(checked ? Qt::Checked : Qt::Unchecked);
}

void CheckableListWidget::clear() {
    for(int i = listWidget->count() - 1; i >= 0; --i) {
        if(listWidget->item(i) != separatorItem) {
            delete listWidget->takeItem(i); // Preserves the separator
        }
    }
    updateHeader();
}

QStringList CheckableListWidget::checkedItems() const {
    QStringList list;
    int sepRow = listWidget->row(separatorItem);
    // We only need to check the clones above the separator
    for (int i = 0; i < sepRow; ++i) {
        list << listWidget->item(i)->text();
    }
    return list;
}

void CheckableListWidget::setCheckedItems(const QStringList &items) {
    isUpdating = true;

    // 1. Purge clones
    int sepRow = listWidget->row(separatorItem);
    for(int i = sepRow - 1; i >= 0; --i) {
        delete listWidget->takeItem(i);
    }

    // 2. Iterate unselected base pool
    sepRow = listWidget->row(separatorItem); // Update row index after deletions
    for (int i = sepRow + 1; i < listWidget->count(); ++i) {
        QListWidgetItem *itm = listWidget->item(i);
        bool checked = items.contains(itm->text());
        itm->setCheckState(checked ? Qt::Checked : Qt::Unchecked);

        if (checked) {
            QListWidgetItem *clone = new QListWidgetItem(itm->text());
            clone->setFlags(Qt::ItemIsUserCheckable | Qt::ItemIsEnabled);
            clone->setCheckState(Qt::Checked);
            listWidget->insertItem(listWidget->row(separatorItem), clone);

            i++; // Offset the index shift caused by the insertion
        }
    }

    isUpdating = false;
    updateHeader();
}

bool CheckableListWidget::isValid() const {
    int checkedCount = listWidget->row(separatorItem);
    if (minChecked > 0 && checkedCount < minChecked) return false;
    if (maxChecked > 0 && checkedCount > maxChecked) return false;
    return true;
}
