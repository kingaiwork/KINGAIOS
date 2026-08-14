import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ColumnLayout {
    id: root
    spacing: 12

    property string runtimeHealth: "offline"
    property string policyMode: "—"
    property int activeTasks: 0
    property int blockedTasks: 0
    property int pendingApprovals: 0
    property bool cloudRequired: false
    property string version: "—"
    property string updatedAt: "—"
    property string state: "waiting"

    function refresh() {
        var xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status !== 0 && xhr.status !== 200) {
                root.runtimeHealth = "offline"
                root.state = "unavailable"
                return
            }
            try {
                var s = JSON.parse(xhr.responseText)
                root.runtimeHealth = s.health || "offline"
                root.policyMode = s.policy || "—"
                root.activeTasks = Number(s.active_tasks || 0)
                root.blockedTasks = Number(s.blocked_tasks || 0)
                root.pendingApprovals = Number(s.pending_approvals || 0)
                root.cloudRequired = s.cloud_required === true
                root.version = s.version || "—"
                root.updatedAt = s.updated_at || "—"
                root.state = "ready"
            } catch (e) {
                root.runtimeHealth = "offline"
                root.state = "invalid"
            }
        }
        xhr.open("GET", "file:///run/kingai/public-status.json")
        xhr.send()
    }

    function attentionText() {
        if (root.runtimeHealth !== "ok") return "Runtime is unavailable. Inspect service health before attempting repair."
        if (root.blockedTasks > 0) return root.blockedTasks + " task(s) are blocked and need investigation."
        if (root.pendingApprovals > 0) return root.pendingApprovals + " approval request(s) are waiting for an authorized decision."
        return "No immediate attention signal is present in the sanitized status snapshot."
    }

    RowLayout {
        Layout.fillWidth: true
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 3
            Label { text: "System Health"; color: "white"; font.pixelSize: 18; font.bold: true }
            Label { text: "Observe first. Repair remains a separately authorized action."; color: "#7f8995"; font.pixelSize: 10 }
        }
        Label {
            text: root.state === "ready" ? "Public status · ready" : "Public status · " + root.state
            color: root.state === "ready" ? "#79d58d" : "#9ba4af"
            font.pixelSize: 10
        }
    }

    GridLayout {
        Layout.fillWidth: true
        columns: 4
        columnSpacing: 10
        rowSpacing: 10

        MetricCard {
            label: "Runtime"
            value: root.runtimeHealth === "ok" ? "Healthy" : "Offline"
            note: root.version
        }
        MetricCard {
            label: "Policy"
            value: root.policyMode
            note: "authority control"
        }
        MetricCard {
            label: "Blocked tasks"
            value: String(root.blockedTasks)
            note: root.blockedTasks > 0 ? "needs attention" : "clear"
        }
        MetricCard {
            label: "Cloud required"
            value: root.cloudRequired ? "Yes" : "No"
            note: root.cloudRequired ? "dependency present" : "local survival path"
        }
    }

    Rectangle {
        Layout.fillWidth: true
        implicitHeight: 122
        radius: 13
        color: "#15191f"
        border.color: root.runtimeHealth === "ok" && root.blockedTasks === 0 ? "#2d4334" : "#55403d"
        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 14
            spacing: 7
            Label { text: "Attention"; color: "white"; font.pixelSize: 13; font.bold: true }
            Label {
                Layout.fillWidth: true
                text: root.attentionText()
                color: "#aeb6c0"
                wrapMode: Text.WordWrap
                font.pixelSize: 11
            }
            Label {
                Layout.fillWidth: true
                text: "This center does not execute repair commands. Privileged remediation must go through Agent identity, Policy, Approval and constrained execution."
                color: "#737d89"
                wrapMode: Text.WordWrap
                font.pixelSize: 9
            }
        }
    }

    RowLayout {
        Layout.fillWidth: true
        Label { text: "Active tasks: " + root.activeTasks; color: "#727c88"; font.pixelSize: 9 }
        Label { text: "Pending approvals: " + root.pendingApprovals; color: "#727c88"; font.pixelSize: 9 }
        Item { Layout.fillWidth: true }
        Label { text: "Updated: " + root.updatedAt; color: "#626c78"; font.pixelSize: 9 }
    }

    Timer {
        interval: 5000
        repeat: true
        running: root.visible
        triggeredOnStart: true
        onTriggered: root.refresh()
    }
}
