import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

ColumnLayout {
    id: root
    spacing: 12

    property string runtimeHealth: "offline"
    property int providerCount: 0
    property string modelMode: "—"
    property string strategy: "—"
    property bool cloudRequired: false
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
                root.providerCount = Number(s.model_providers || 0)
                root.modelMode = s.model_mode || "—"
                root.strategy = s.model_strategy || "—"
                root.cloudRequired = s.cloud_required === true
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

    RowLayout {
        Layout.fillWidth: true
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 3
            Label { text: "Model Fabric"; color: "white"; font.pixelSize: 18; font.bold: true }
            Label { text: "Provider-neutral status without exposing credentials or provider-private configuration."; color: "#7f8995"; font.pixelSize: 10 }
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
            label: "Providers"
            value: root.providerCount > 0 ? String(root.providerCount) : "—"
            note: root.providerCount > 0 ? "configured" : "not configured"
        }
        MetricCard {
            label: "Mode"
            value: root.modelMode
            note: "routing posture"
        }
        MetricCard {
            label: "Strategy"
            value: root.strategy
            note: "provider policy"
        }
        MetricCard {
            label: "Cloud required"
            value: root.cloudRequired ? "Yes" : "No"
            note: root.cloudRequired ? "external dependency" : "local survival path"
        }
    }

    Rectangle {
        Layout.fillWidth: true
        implicitHeight: 118
        radius: 13
        color: "#15191f"
        border.color: "#282e37"
        ColumnLayout {
            anchors.fill: parent
            anchors.margins: 14
            spacing: 7
            Label { text: "Routing boundary"; color: "white"; font.pixelSize: 13; font.bold: true }
            Label {
                Layout.fillWidth: true
                text: root.providerCount > 0
                      ? "KINGAI Model Fabric has configured providers. Selection still happens through the governed model router; this view does not contain API keys, endpoint secrets or reusable provider credentials."
                      : "No providers are currently represented in the sanitized status snapshot. KINGAI OS core Policy, Task, Memory and Audit remain independent of a cloud-model survival dependency."
                color: "#9ca6b2"
                wrapMode: Text.WordWrap
                font.pixelSize: 11
            }
        }
    }

    Label {
        Layout.fillWidth: true
        text: "Last public status update: " + root.updatedAt
        color: "#626c78"
        font.pixelSize: 9
    }

    Timer {
        interval: 5000
        repeat: true
        running: root.visible
        triggeredOnStart: true
        onTriggered: root.refresh()
    }
}
