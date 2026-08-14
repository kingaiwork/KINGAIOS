import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ColumnLayout {
    id: root
    spacing: 10

    property var tasks: []
    property string bridgeState: "waiting"
    property string bridgeUpdatedAt: "—"
    property string bridgeError: ""

    function privateStatusUrl() {
        var runtime = StandardPaths.writableLocation(StandardPaths.RuntimeLocation)
        if (!runtime || runtime.length === 0) return ""
        return "file://" + runtime + "/kingai/desktop-private.json"
    }

    function refresh() {
        var url = privateStatusUrl()
        if (url === "") {
            root.bridgeState = "unavailable"
            root.bridgeError = "Runtime directory unavailable"
            root.tasks = []
            return
        }
        var xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status !== 0 && xhr.status !== 200) {
                root.bridgeState = "waiting"
                root.bridgeError = "Private bridge snapshot not ready"
                root.tasks = []
                return
            }
            try {
                var snapshot = JSON.parse(xhr.responseText)
                if (snapshot.schema !== 1 || snapshot.product !== "KINGAI OS Desktop" || !(snapshot.tasks instanceof Array)) {
                    throw new Error("invalid private snapshot schema")
                }
                var updated = Date.parse(snapshot.updated_at || "")
                if (isNaN(updated)) throw new Error("invalid private snapshot timestamp")
                var ageMs = Date.now() - updated
                if (ageMs < -60000) throw new Error("private snapshot timestamp is in the future")

                root.tasks = snapshot.tasks
                root.bridgeUpdatedAt = snapshot.updated_at || "—"
                if (ageMs > 15000) {
                    root.bridgeState = "stale"
                    root.bridgeError = "Private bridge data is stale; showing last known tasks"
                } else {
                    root.bridgeState = "ready"
                    root.bridgeError = ""
                }
            } catch (e) {
                root.bridgeState = "invalid"
                root.bridgeError = "Private bridge snapshot rejected"
                root.tasks = []
            }
        }
        xhr.open("GET", url)
        xhr.send()
    }

    RowLayout {
        Layout.fillWidth: true
        Label {
            Layout.fillWidth: true
            text: "My Tasks"
            color: "white"
            font.pixelSize: 18
            font.bold: true
        }
        Label {
            text: "Private bridge · " + root.bridgeState
            color: root.bridgeState === "ready" ? "#79d58d" : (root.bridgeState === "stale" ? "#d4aa68" : "#9ba4af")
            font.pixelSize: 10
        }
    }

    Label {
        Layout.fillWidth: true
        text: "This list is private to your login UID. The bridge strips step targets, capabilities, approval IDs, results and errors before writing the 0600 desktop snapshot."
        color: "#7f8995"
        wrapMode: Text.WordWrap
        font.pixelSize: 10
    }

    Label {
        Layout.fillWidth: true
        visible: root.bridgeState !== "ready" && root.bridgeError !== ""
        text: root.bridgeError
        color: root.bridgeState === "stale" ? "#d4aa68" : "#9099a5"
        font.pixelSize: 12
    }

    Label {
        Layout.fillWidth: true
        visible: (root.bridgeState === "ready" || root.bridgeState === "stale") && root.tasks.length === 0
        text: "No tasks for this user yet."
        color: "#9099a5"
        font.pixelSize: 12
    }

    Repeater {
        model: root.tasks
        delegate: Rectangle {
            required property var modelData
            Layout.fillWidth: true
            implicitHeight: 86
            radius: 13
            color: "#15191f"
            border.color: "#282e37"

            RowLayout {
                anchors.fill: parent
                anchors.margins: 14
                spacing: 14

                Rectangle {
                    implicitWidth: 10
                    Layout.fillHeight: true
                    radius: 5
                    color: modelData.status === "running" ? "#447b55" : (modelData.status === "blocked" || modelData.status === "failed" ? "#8b4c48" : (modelData.status === "waiting_approval" ? "#8b6c3f" : "#495461"))
                }

                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: 4
                    Label {
                        Layout.fillWidth: true
                        text: modelData.goal || "Untitled task"
                        color: "white"
                        font.pixelSize: 14
                        font.bold: true
                        elide: Text.ElideRight
                    }
                    Label {
                        Layout.fillWidth: true
                        text: (modelData.agent || "main") + "  ·  " + (modelData.status || "unknown")
                        color: "#9ca6b2"
                        font.pixelSize: 11
                        elide: Text.ElideRight
                    }
                    Label {
                        text: "Steps " + (modelData.done_steps || 0) + "/" + (modelData.step_count || 0) + ((modelData.failed_steps || 0) > 0 ? "  ·  " + modelData.failed_steps + " blocked/failed" : "")
                        color: "#737d89"
                        font.pixelSize: 10
                    }
                }

                Label {
                    text: modelData.updated_at || "—"
                    color: "#68727e"
                    font.pixelSize: 9
                    Layout.preferredWidth: 180
                    horizontalAlignment: Text.AlignRight
                    elide: Text.ElideRight
                }
            }
        }
    }

    Label {
        Layout.fillWidth: true
        visible: root.bridgeState === "ready" || root.bridgeState === "stale"
        text: "Private snapshot updated: " + root.bridgeUpdatedAt
        color: "#626c78"
        font.pixelSize: 9
    }

    Timer {
        interval: 3000
        repeat: true
        running: root.visible
        triggeredOnStart: true
        onTriggered: root.refresh()
    }
}
