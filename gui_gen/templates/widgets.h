// Custom widget classes
class AppComboBox : public QComboBox {
public:
    using QComboBox::QComboBox;

protected:
    void showPopup() override;
    void hidePopup() override;
    void paintEvent(QPaintEvent *event) override;

private:
    bool m_popupOpen = false;
};

class AppDialog : public QDialog {
public:
    explicit AppDialog(QWidget *parent = nullptr);
    ~AppDialog() override;

    QVBoxLayout *contentLayout() const;

protected:
    bool event(QEvent *event) override;
    bool eventFilter(QObject *watched, QEvent *event) override;

private:
    Qt::Edges resizeEdges(const QPoint &pos) const;
    void setCursorForEdges(Qt::Edges edges);
    void toggleMaximized();

    QLabel *m_titleLabel = nullptr;
    QVBoxLayout *m_contentLayout = nullptr;
    QPushButton *m_maxButton = nullptr;
};

class InternalListWidget : public QListWidget {
protected:
    void mouseReleaseEvent(QMouseEvent *event) override;
public:
    InternalListWidget(QWidget *parent = nullptr) : QListWidget(parent) {}
};

class CheckableListWidget : public QWidget {
public:
    QLineEdit *searchBox;
    QWidget *headerWidget;
    QLabel *statusLabel;
    QPushButton *toggleAllBtn;
    InternalListWidget *listWidget;
    QListWidgetItem *separatorItem;

    int minChecked = 0;
    int maxChecked = 0; // 0 implies no limit
    bool isUpdating = false;

    CheckableListWidget(QWidget *parent = nullptr);
    void updateHeader();
    void addItem(const QString &text, bool checked = false);
    void addItem(const QString &text, const QString &userData, bool checked = false);
    void clear();
    QStringList checkedItems() const;
    void setCheckedItems(const QStringList &items);
    bool isValid() const;
};
