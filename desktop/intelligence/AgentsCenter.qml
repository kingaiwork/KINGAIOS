import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ColumnLayout {
    id: root
    spacing: 10

    property var agents: []
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
            root.agents = []
            return
        }
        var xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status !== 0 && xhr.status !== 200) {
                root.bridgeState = "waiting"
                root.bridgeError = "Private bridge snapshot not ready"
                root.agents = []
                return
            }
            try {
                var snapshot = JSON.parse(xhr.responseText)
                if (snapshot.schema !== 1 || snapshot.product !== "KINGAI OS Desktop" || !(snapshot.agents instanceof Array)) throw new Error("invalid private snapshot schema")
                var updated = Date.parse(snapshot.updated_at || "")
                if (isNaN(updated)) throw new Error("invalid private snapshot timestamp")
                var ageMs = Date.now() - updated
                if (ageMs < -60000) throw new Error("private snapshot timestamp is in the future")
                root.agents = snapshot.agents
                root.bridgeUpdatedAt = snapshot.updated_at || "—"
                if (ageMs > 15000) {
                    root.bridgeState = "stale"
                    root.bridgeError = "Agent metadata is stale; showing last known authorization state"
                } else {
                    root.bridgeState = "ready"
                    root.bridgeError = ""
                }
            } catch (e) {
                root.bridgeState = "invalid"
                root.bridgeError = "Private Agent metadata rejected"
                root.agents = []
            }
        }
        xhr.open("GET", url)
        xhr.send()
    }

    RowLayout {
        Layout.fillWidth: true
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 3
            Label { text: "My Agent Access"; color: "white"; font.pixelSize: 18; font.bold: true }
            Label { text: "Visibility does not grant authority. Identity and Policy still decide what can run."; color: "#7f8995"; font.pixelSize: 10 }
        }
        Label {
            text: "Private bridge · " + root.bridgeState
            color: root.bridgeState === "ready" ? "#79d58d" : (root.bridgeState === "stale" ? "#d4aa68" : "#9ba4af")
            font.pixelSize: 10
        }
    }

    Label {
        Layout.fillWidth: true
        visible: root.bridgeError !== ""
        text: root.bridgeError
        color: root.bridgeState === "stale" ? "#d4aa68" : "#9099a5"
        font.pixelSize: 11
    }

    Label {
        Layout.fillWidth: true
        visible: (root.bridgeState === "ready" || root.bridgeState === "stale") && root.agents.length === 0
        text: "No Agent definitions are available."
        color: "#9099a5"
        font.pixelSize: 12
    }

    Repeater {
        model: root.agents
        delegate: Rectangle {
            required property var modelData
            Layout.fillWidth: true
            implicitHeight: 88
            radius: 13
            color: "#15191f"
            border.color: modelData.authorized_for_peer === true ? "#315b3c" : "#353b44"

            RowLayout {
                anchors.fill: parent
                anchors.margins: 14
                spacing: 14

                Rectangle {
                    implicitWidth: 42
                    implicitHeight: 42
                    radius: 21
                    color: modelData.authorized_for_peer === true ? "#1c3a26" : "#22272e"
                    Label {
                        anchors.centerIn: parent
                        text: String(modelData.id || "A").substring(0, 1).toUpperCase()
                        color: "white"
                        font.pixelSize: 16
                        font.bold: true
                    }
                }

                ColumnLayout {
                    Layout.fillWidth: true
                    spacing: 4
                    Label {
                        Layout.fillWidth: true
                        text: modelData.id || "unknown"
                        color: "white"
                        font.pixelSize: 14
                        font.bold: true
                        elide: Text.ElideRight
                    }
                    Label {
                        Layout.fillWidth: true
                        text: (modelData.role || "unspecified") + "  ·  " + Number(modelData.capability_count || 0) + " declared capability class(es)"
                        color: "#8f99a5"
                        font.pixelSize: 10
                        elide: Text.ElideRight
                    }
                }

                Rectangle {
                    implicitWidth: 104
                    implicitHeight: 30
                    radius: 15
                    color: modelData.authorized_for_peer === true ? "#183821" : "#282d34"
                    Label {
                        anchors.centerIn: parent
                        text: modelData.authorized_for_peer === true ? "AUTHORIZED" : "LOCKED"
                        color: modelData.authorized_for_peer === true ? "#9ee7ad" : "#abb2bb"
                        font.pixelSize: 9
                        font.bold: true
                    }
                }
            }
        }
    }

    Label {
        Layout.fillWidth: true
        text: "Capability names are intentionally not included in the Desktop snapshot. Actual actions are re-evaluated by Agent identity, capability declaration, Policy and Approval at execution time."
        color: "#68727e"
        wrapMode: Text.WordWrap
        font.pixelSize: 9
    }

    Label {
        Layout.fillWidth: true
        visible: root.bridgeState === "ready" || root.bridgeState === "stale"
        text: "Private snapshot updated: " + root.bridgeUpdatedAt
        color: "#626c78"
        font.pixelSize: 9
    }

    Timer {
        interval: 4000
        repeat: true
        running: root.visible
        triggeredOnStart: true
        onTriggered: root.refresh()
    }
}
