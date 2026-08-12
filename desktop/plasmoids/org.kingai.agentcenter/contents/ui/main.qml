import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import org.kde.plasma.plasmoid

PlasmoidItem {
    id: root
    property string health: "offline"
    property string version: "—"
    property int registeredAgents: 0
    property string modelMode: "—"
    property string memoryMode: "—"
    property string updatedAt: "—"

    toolTipMainText: "KINGAI Agent Center"
    toolTipSubText: health === "ok" ? "KINGAI Core online" : "KINGAI Core status unavailable"

    compactRepresentation: ToolButton {
        text: "K"
        font.bold: true
        Accessible.name: "KINGAI Agent Center"
        onClicked: root.expanded = !root.expanded
    }

    fullRepresentation: ColumnLayout {
        spacing: 10
        implicitWidth: 300
        Label { text: "KINGAI Agent Center"; font.bold: true; font.pixelSize: 18 }
        RowLayout {
            Label { text: "Core"; Layout.fillWidth: true }
            Label { text: root.health === "ok" ? "Online" : "Offline"; font.bold: true }
        }
        RowLayout {
            Label { text: "Version"; Layout.fillWidth: true }
            Label { text: root.version }
        }
        RowLayout {
            Label { text: "Registered agents"; Layout.fillWidth: true }
            Label { text: String(root.registeredAgents) }
        }
        RowLayout {
            Label { text: "Model mode"; Layout.fillWidth: true }
            Label { text: root.modelMode }
        }
        RowLayout {
            Label { text: "Memory"; Layout.fillWidth: true }
            Label { text: root.memoryMode }
        }
        Label {
            text: "Local status only · no prompts, secrets or memory content"
            wrapMode: Text.WordWrap
            opacity: 0.7
            Layout.fillWidth: true
        }
        Label { text: "Updated: " + root.updatedAt; opacity: 0.6; font.pixelSize: 11 }
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

    Timer { interval: 5000; repeat: true; running: true; triggeredOnStart: true; onTriggered: root.refreshStatus() }
}
