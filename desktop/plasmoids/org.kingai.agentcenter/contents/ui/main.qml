import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.plasma.plasmoid

PlasmoidItem {
    id: root

    property string health: "offline"
    property string version: "—"
    property int registeredAgents: 0
    property int activeTasks: 0
    property int pendingApprovals: 0
    property int modelProviders: 0
    property string modelMode: "—"
    property string memoryMode: "—"
    property string updatedAt: "—"

    toolTipMainText: "KINGAI Center"
    toolTipSubText: health === "ok" ? "Local KINGAI Runtime online" : "KINGAI Runtime status unavailable"

    compactRepresentation: ToolButton {
        text: "K"
        font.bold: true
        Accessible.name: "KINGAI Center"
        onClicked: root.expanded = !root.expanded
    }

    fullRepresentation: ColumnLayout {
        spacing: 12
        implicitWidth: 360

        RowLayout {
            Layout.fillWidth: true
            ColumnLayout {
                spacing: 2
                Layout.fillWidth: true
                Label { text: "KINGAI Center"; font.bold: true; font.pixelSize: 19 }
                Label { text: "Private local system overview"; opacity: 0.65; font.pixelSize: 11 }
            }
            Label {
                text: root.health === "ok" ? "● Online" : "● Offline"
                font.bold: true
                color: root.health === "ok" ? "#248A3D" : "#B3261E"
            }
        }

        GridLayout {
            columns: 2
            columnSpacing: 8
            rowSpacing: 8
            Layout.fillWidth: true

            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    Label { text: "Runtime"; opacity: 0.65; font.pixelSize: 11 }
                    Label { text: root.health === "ok" ? "Healthy" : "Offline"; font.bold: true; font.pixelSize: 16 }
                    Label { text: root.version; opacity: 0.7; font.pixelSize: 10 }
                }
            }
            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    Label { text: "Agents"; opacity: 0.65; font.pixelSize: 11 }
                    Label { text: String(root.registeredAgents); font.bold: true; font.pixelSize: 20 }
                    Label { text: "registered"; opacity: 0.7; font.pixelSize: 10 }
                }
            }
            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    Label { text: "Tasks"; opacity: 0.65; font.pixelSize: 11 }
                    Label { text: String(root.activeTasks); font.bold: true; font.pixelSize: 20 }
                    Label { text: "active"; opacity: 0.7; font.pixelSize: 10 }
                }
            }
            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    Label { text: "Approvals"; opacity: 0.65; font.pixelSize: 11 }
                    Label {
                        text: String(root.pendingApprovals)
                        font.bold: true
                        font.pixelSize: 20
                        color: root.pendingApprovals > 0 ? "#A15C00" : palette.text
                    }
                    Label { text: root.pendingApprovals > 0 ? "pending" : "clear"; opacity: 0.7; font.pixelSize: 10 }
                }
            }
            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    Label { text: "Memory"; opacity: 0.65; font.pixelSize: 11 }
                    Label { text: root.memoryMode; font.bold: true; font.pixelSize: 15 }
                    Label { text: "content stays private"; opacity: 0.7; font.pixelSize: 10 }
                }
            }
            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    Label { text: "Models"; opacity: 0.65; font.pixelSize: 11 }
                    Label { text: root.modelProviders > 0 ? String(root.modelProviders) : "None"; font.bold: true; font.pixelSize: 18 }
                    Label { text: root.modelProviders > 0 ? root.modelMode : "not configured"; opacity: 0.7; font.pixelSize: 10 }
                }
            }
        }

        Button {
            Layout.fillWidth: true
            text: "Open KINGAI Intelligence"
            onClicked: Qt.openUrlExternally("kingai://home")
        }

        RowLayout {
            Layout.fillWidth: true
            Button {
                Layout.fillWidth: true
                text: root.pendingApprovals > 0 ? "Approvals (" + root.pendingApprovals + ")" : "Approvals"
                onClicked: Qt.openUrlExternally("kingai://approvals")
            }
            Button {
                Layout.fillWidth: true
                text: "Tasks"
                onClicked: Qt.openUrlExternally("kingai://tasks")
            }
        }

        Label {
            text: "Counts only · no prompts, task goals, targets, secrets or memory content"
            wrapMode: Text.WordWrap
            opacity: 0.65
            font.pixelSize: 10
            Layout.fillWidth: true
        }
        Label {
            text: "Updated: " + root.updatedAt
            opacity: 0.5
            font.pixelSize: 10
            Layout.fillWidth: true
        }
    }

    function refreshStatus() {
        var xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status !== 0 && xhr.status !== 200) {
                root.health = "offline"
                return
            }
            try {
                var s = JSON.parse(xhr.responseText)
                root.health = s.health || "offline"
                root.version = s.version || "—"
                root.registeredAgents = s.registered_agents || 0
                root.activeTasks = s.active_tasks || 0
                root.pendingApprovals = s.pending_approvals || 0
                root.modelProviders = s.model_providers || 0
                root.modelMode = s.model_mode || "—"
                root.memoryMode = s.memory_mode || "—"
                root.updatedAt = s.updated_at || "—"
            } catch (e) {
                root.health = "offline"
            }
        }
        xhr.open("GET", "file:///run/kingai/public-status.json")
        xhr.send()
    }

    Timer {
        interval: 5000
        repeat: true
        running: true
        triggeredOnStart: true
        onTriggered: root.refreshStatus()
    }
}
