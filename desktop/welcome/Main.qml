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
    color: "#F4F0E9"

    property string selected: desktopSettings.experience
    property color ink: "#23252A"
    property color muted: "#6B6F76"
    property color card: "#FFFCF8"
    property color line: "#DDD7CE"
    property color accent: "#4E67A6"

    Settings {
        id: desktopSettings
        location: StandardPaths.writableLocation(StandardPaths.ConfigLocation) + "/kingai-desktop.ini"
        property string experience: ""
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 44
        spacing: 18

        RowLayout {
            Layout.fillWidth: true
            spacing: 16

            Rectangle {
                width: 42
                height: 42
                radius: 13
                color: "#2C3139"
                Label {
                    anchors.centerIn: parent
                    text: "K"
                    color: "white"
                    font.bold: true
                    font.pixelSize: 21
                }
            }

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 2
                Label { text: "Welcome to KINGAI OS"; color: root.ink; font.bold: true; font.pixelSize: 30 }
                Label { text: "Choose how your desktop should feel. The same governed runtime stays underneath."; color: root.muted; font.pixelSize: 15 }
            }

            Rectangle {
                radius: 12
                color: "#E8EFE7"
                border.color: "#CAD8C9"
                implicitWidth: privacyLabel.implicitWidth + 24
                implicitHeight: 34
                Label {
                    id: privacyLabel
                    anchors.centerIn: parent
                    text: "Local-first by default"
                    color: "#405345"
                    font.pixelSize: 12
                    font.bold: true
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 16

            Repeater {
                model: [
                    { id: "kingai-intelligence", title: "KINGAI Intelligence", note: "AI-first workspace for agents, tasks, memory and knowledge.", tone: "#E7EDF7", preview: "Agents   Tasks\n\n   Intelligence\n\nMemory   Models" },
                    { id: "kingai-flow", title: "KINGAI Flow", note: "Calm dock-oriented desktop that keeps the workspace visually quiet.", tone: "#E9EEE9", preview: "       Workspace\n\n\n  ●  ●  K  ●  ●\n        Dock" },
                    { id: "kingai-classic", title: "KINGAI Classic", note: "Familiar taskbar and app-menu workflow for an easy PC transition.", tone: "#EEEAE5", preview: "Desktop\n\n\n▣  Apps        System  ◷" }
                ]

                delegate: Rectangle {
                    required property var modelData
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: 20
                    color: root.selected === modelData.id ? "#FFFFFF" : root.card
                    border.width: root.selected === modelData.id ? 2 : 1
                    border.color: root.selected === modelData.id ? root.accent : root.line

                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 18
                        spacing: 12

                        Rectangle {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 245
                            radius: 16
                            color: modelData.tone
                            border.color: "#D8D4CE"

                            Label {
                                anchors.centerIn: parent
                                text: modelData.preview
                                color: "#343941"
                                horizontalAlignment: Text.AlignHCenter
                                font.pixelSize: 18
                                lineHeight: 1.25
                            }

                            Rectangle {
                                visible: modelData.id === "kingai-flow"
                                anchors.top: parent.top
                                anchors.right: parent.right
                                anchors.margins: 12
                                radius: 10
                                color: "#FFFFFFCC"
                                implicitWidth: recommended.implicitWidth + 18
                                implicitHeight: 28
                                Label {
                                    id: recommended
                                    anchors.centerIn: parent
                                    text: "Recommended"
                                    color: "#4B5B50"
                                    font.pixelSize: 11
                                    font.bold: true
                                }
                            }
                        }

                        Label { text: modelData.title; color: root.ink; font.bold: true; font.pixelSize: 20 }
                        Label {
                            text: modelData.note
                            color: root.muted
                            font.pixelSize: 13
                            wrapMode: Text.WordWrap
                            Layout.fillWidth: true
                        }

                        Item { Layout.fillHeight: true }

                        Button {
                            id: chooseButton
                            Layout.fillWidth: true
                            text: root.selected === modelData.id ? "Selected" : "Choose this experience"
                            onClicked: root.selected = modelData.id
                            contentItem: Label {
                                text: chooseButton.text
                                color: root.selected === modelData.id ? "white" : root.ink
                                horizontalAlignment: Text.AlignHCenter
                                verticalAlignment: Text.AlignVCenter
                                font.pixelSize: 13
                                font.bold: true
                            }
                            background: Rectangle {
                                radius: 12
                                color: root.selected === modelData.id ? root.accent : "#F2EEE8"
                                border.color: root.selected === modelData.id ? root.accent : root.line
                            }
                        }
                    }
                }
            }
        }

        RowLayout {
            Layout.fillWidth: true
            spacing: 14

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 2
                Label {
                    text: root.selected === "" ? "Choose an experience to continue." : "Ready: " + root.selected
                    color: root.ink
                    font.pixelSize: 13
                    font.bold: true
                }
                Label {
                    text: "You can switch later from Settings or the KINGAI CLI. This choice changes layout, not the system security model."
                    color: root.muted
                    font.pixelSize: 11
                }
            }

            Button {
                id: continueButton
                text: "Continue"
                enabled: root.selected !== ""
                implicitWidth: 140
                implicitHeight: 44
                onClicked: {
                    desktopSettings.experience = root.selected
                    desktopSettings.sync()
                    Qt.quit()
                }
                contentItem: Label {
                    text: continueButton.text
                    color: continueButton.enabled ? "white" : "#8B8E94"
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                    font.pixelSize: 14
                    font.bold: true
                }
                background: Rectangle {
                    radius: 13
                    color: continueButton.enabled ? root.accent : "#DEDAD3"
                }
            }
        }
    }
}
