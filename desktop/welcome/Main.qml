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
    title: "KINGAI Welcome"
    color: "#111418"
    property string selected: desktopSettings.experience

    Settings {
        id: desktopSettings
        location: StandardPaths.writableLocation(StandardPaths.ConfigLocation) + "/kingai-desktop.ini"
        property string experience: ""
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 48
        spacing: 24
        Label { text: "Choose your KINGAI desktop experience"; color: "white"; font.pixelSize: 30 }
        Label { text: "Preview each experience now. You can change it later in Settings or with the KINGAI CLI."; color: "#b8c0cc"; font.pixelSize: 16 }
        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 18
            Repeater {
                model: [
                    { id: "kingai-intelligence", title: "KINGAI Intelligence", note: "AI-first · Agents · Memory · Knowledge", preview: "Agents   Memory\n\n        AI Workspace\n\nTasks    Knowledge" },
                    { id: "kingai-flow", title: "KINGAI Flow", note: "Dock-oriented · Spatial · Clean", preview: "       Workspace\n\n\n  ◉  ◉  ◉  ◉  ◉\n        Dock" },
                    { id: "kingai-classic", title: "KINGAI Classic", note: "Taskbar · App menu · Familiar PC workflow", preview: "Desktop\n\n\n▣  Apps        System  ◷" }
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
                            Label { anchors.centerIn: parent; text: modelData.preview; color: "white"; horizontalAlignment: Text.AlignHCenter; font.pixelSize: 20 }
                        }
                        Label { text: modelData.title; color: "white"; font.pixelSize: 21 }
                        Label { text: modelData.note; color: "#aeb7c2"; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                        Item { Layout.fillHeight: true }
                        Button { text: root.selected === modelData.id ? "Selected" : "Preview & choose"; onClicked: root.selected = modelData.id; Layout.fillWidth: true }
                    }
                }
            }
        }
        RowLayout {
            Layout.fillWidth: true
            Label { Layout.fillWidth: true; text: root.selected === "" ? "Select an experience to continue." : "Selected: " + root.selected; color: "#b8c0cc" }
            Button {
                text: "Continue"
                enabled: root.selected !== ""
                onClicked: {
                    desktopSettings.experience = root.selected
                    desktopSettings.sync()
                    Qt.quit()
                }
            }
        }
    }
}
