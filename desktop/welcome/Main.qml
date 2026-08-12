import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ApplicationWindow {
    width: 1080
    height: 700
    visible: true
    title: "KINGAI Welcome"
    color: "#111418"
    property string selected: ""

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 48
        spacing: 24
        Label { text: "Choose your KINGAI desktop experience"; color: "white"; font.pixelSize: 30 }
        Label { text: "Preview each experience now. You can change it later in Settings."; color: "#b8c0cc"; font.pixelSize: 16 }
        RowLayout {
            Layout.fillWidth: true
            Layout.fillHeight: true
            spacing: 18
            Repeater {
                model: [
                    { id: "kingai-intelligence", title: "KINGAI Intelligence", note: "AI-first · Agents · Memory · Knowledge" },
                    { id: "kingai-flow", title: "KINGAI Flow", note: "Dock-oriented · Spatial · Clean" },
                    { id: "kingai-classic", title: "KINGAI Classic", note: "Taskbar · App menu · Familiar PC workflow" }
                ]
                delegate: Rectangle {
                    required property var modelData
                    Layout.fillWidth: true
                    Layout.fillHeight: true
                    radius: 18
                    color: selected === modelData.id ? "#253041" : "#191e25"
                    border.color: selected === modelData.id ? "#8fb7ff" : "#343b45"
                    ColumnLayout {
                        anchors.fill: parent
                        anchors.margins: 22
                        spacing: 16
                        Rectangle {
                            Layout.fillWidth: true
                            Layout.preferredHeight: 250
                            radius: 12
                            color: modelData.id === "kingai-intelligence" ? "#0d1825" : (modelData.id === "kingai-flow" ? "#18211f" : "#202126")
                            Label { anchors.centerIn: parent; text: modelData.title + "\nPreview"; color: "white"; horizontalAlignment: Text.AlignHCenter; font.pixelSize: 22 }
                        }
                        Label { text: modelData.title; color: "white"; font.pixelSize: 21 }
                        Label { text: modelData.note; color: "#aeb7c2"; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                        Item { Layout.fillHeight: true }
                        Button { text: selected === modelData.id ? "Selected" : "Choose"; onClicked: selected = modelData.id; Layout.fillWidth: true }
                    }
                }
            }
        }
        Label { text: selected === "" ? "Select an experience to continue." : "Selected: " + selected + " — apply with: kingai desktop set " + selected; color: "#b8c0cc" }
    }
}
