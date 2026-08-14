import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: root
    width: 1080
    height: 700
    minimumWidth: 860
    minimumHeight: 600
    visible: true
    title: "KINGAI OS Desktop"
    color: "#111418"
    property string selected: desktopSettings.experience === "" ? "kingai-intelligence" : desktopSettings.experience

    Settings {
        id: desktopSettings
        location: StandardPaths.writableLocation(StandardPaths.ConfigLocation) + "/kingai-desktop.ini"
        property string experience: ""
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 48
        spacing: 24

        Label {
            text: "Welcome to KINGAI OS Desktop"
            color: "white"
            font.pixelSize: 30
        }
        Label {
            text: "Desktop is the personal-computer / PC edition. Choose the experience that fits how you work; the same governed KINGAI Core stays underneath."
            color: "#b8c0cc"
            font.pixelSize: 16
            wrapMode: Text.WordWrap
            Layout.fillWidth: true
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 18

            Repeater {
                model: [
                    {
                        id: "kingai-intelligence",
                        title: "KINGAI Intelligence",
                        note: "Recommended · AI-first · Agents · Tasks · Memory",
                        preview: "Agents   Memory\n\n        AI Workspace\n\nTasks    Knowledge"
                    },
                    {
                        id: "kingai-flow",
                        title: "KINGAI Flow",
                        note: "Modern · Dock-oriented · Workspace-first",
                        preview: "       Workspace\n\n\n  ◉  ◉  ◉  ◉  ◉\n        Dock"
                    },
                    {
                        id: "kingai-classic",
                        title: "KINGAI Classic",
                        note: "Familiar · Taskbar · App menu · System tray",
                        preview: "Desktop\n\n\n▣  Apps        System  ◷"
                    }
                ]

                delegate: Rectangle {
                    required property var modelData
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: 18
                    color: root.selected === modelData.id ? "#253041" : "#191e25"
                    border.color: root.selected === modelData.id ? "#8fb7ff" : "#343b45"

                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 22
                        spacing: 16

                        Rectangle {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 250
                            radius: 12
                            color: modelData.id === "kingai-intelligence" ? "#0d1825" : (modelData.id === "kingai-flow" ? "#18211f" : "#202126")
                            Label {
                                anchors.centerIn: parent
                                text: modelData.preview
                                color: "white"
                                horizontalAlignment: Text.AlignHCenter
                                font.pixelSize: 20
                            }
                        }

                        Label {
                            text: modelData.title
                            color: "white"
                            font.pixelSize: 21
                        }
                        Label {
                            text: modelData.note
                            color: "#aeb7c2"
                            wrapMode: Text.WordWrap
                            Layout.fillWidth: true
                        }
                        Item { Layout.fillHeight: true }
                        Button {
                            text: root.selected === modelData.id ? "Selected" : "Choose"
                            onClicked: root.selected = modelData.id
                            Layout.fillWidth: true
                        }
                    }
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Label {
                Layout.fillWidth: true
                text: "Selected: " + root.selected
                color: "#b8c0cc"
            }
            Button {
                text: "Enter Desktop"
                onClicked: {
                    desktopSettings.experience = root.selected
                    desktopSettings.sync()
                    Qt.quit()
                }
            }
        }
    }
}
