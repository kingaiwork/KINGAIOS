import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ColumnLayout {
    id: root
    spacing: 12

    property int total: 0
    property int expiring: 0
    property var byLayer: ({})
    property var bySensitivity: ({})
    property string bridgeState: "waiting"
    property string bridgeUpdatedAt: "—"
    property string bridgeError: ""

    readonly property var layers: [
        { id: "M0", label: "Context" },
        { id: "M1", label: "Working" },
        { id: "M2", label: "Task" },
        { id: "M3", label: "Episodic" },
        { id: "M4", label: "Semantic" },
        { id: "M5", label: "User / Org" },
        { id: "M6", label: "Evolution" }
    ]

    function privateStatusUrl() {
        var runtime = StandardPaths.writableLocation(StandardPaths.RuntimeLocation)
        if (!runtime || runtime.length === 0) return ""
        return "file://" + runtime + "/kingai/desktop-private.json"
    }

    function layerCount(layer) {
        return root.byLayer && root.byLayer[layer] !== undefined ? Number(root.byLayer[layer]) : 0
    }

    function sensitivitySummary() {
        if (!root.bySensitivity) return "No memory metadata"
        var keys = Object.keys(root.bySensitivity).sort()
        if (keys.length === 0) return "No memory metadata"
        var parts = []
        for (var i = 0; i < keys.length; ++i) parts.push(keys[i] + " " + root.bySensitivity[keys[i]])
        return parts.join("  ·  ")
    }

    function refresh() {
        var url = privateStatusUrl()
        if (url === "") {
            root.bridgeState = "unavailable"
            root.bridgeError = "Runtime directory unavailable"
            return
        }
        var xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status !== 0 && xhr.status !== 200) {
                root.bridgeState = "waiting"
                root.bridgeError = "Private bridge snapshot not ready"
                return
            }
            try {
                var snapshot = JSON.parse(xhr.responseText)
                if (snapshot.schema !== 1 || snapshot.product !== "KINGAI OS Desktop" || !snapshot.memory) throw new Error("invalid private snapshot schema")
                var updated = Date.parse(snapshot.updated_at || "")
                if (isNaN(updated)) throw new Error("invalid private snapshot timestamp")
                var ageMs = Date.now() - updated
                if (ageMs < -60000) throw new Error("private snapshot timestamp is in the future")

                root.total = Number(snapshot.memory.total || 0)
                root.expiring = Number(snapshot.memory.expiring || 0)
                root.byLayer = snapshot.memory.by_layer || ({})
                root.bySensitivity = snapshot.memory.by_sensitivity || ({})
                root.bridgeUpdatedAt = snapshot.updated_at || "—"
                if (ageMs > 15000) {
                    root.bridgeState = "stale"
                    root.bridgeError = "Memory summary is stale; showing last known metadata counts"
                } else {
                    root.bridgeState = "ready"
                    root.bridgeError = ""
                }
            } catch (e) {
                root.bridgeState = "invalid"
                root.bridgeError = "Private Memory summary rejected"
                root.total = 0
                root.expiring = 0
                root.byLayer = ({})
                root.bySensitivity = ({})
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
            Label { text: "My Memory"; color: "white"; font.pixelSize: 18; font.bold: true }
            Label { text: "Metadata overview only · Memory Data never enters this desktop snapshot"; color: "#7f8995"; font.pixelSize: 10 }
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

    RowLayout {
        Layout.fillWidth: true
        spacing: 12
        Rectangle {
            Layout.preferredWidth: 170
            implicitHeight: 86
            radius: 13
            color: "#15191f"
            border.color: "#282e37"
            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 14
                Label { text: "Total records"; color: "#87919d"; font.pixelSize: 10 }
                Item { Layout.fillHeight: true }
                Label { text: String(root.total); color: "white"; font.pixelSize: 24; font.bold: true }
            }
        }
        Rectangle {
            Layout.preferredWidth: 170
            implicitHeight: 86
            radius: 13
            color: "#15191f"
            border.color: "#282e37"
            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 14
                Label { text: "Expiring"; color: "#87919d"; font.pixelSize: 10 }
                Item { Layout.fillHeight: true }
                Label { text: String(root.expiring); color: "white"; font.pixelSize: 24; font.bold: true }
            }
        }
        ColumnLayout {
            Layout.fillWidth: true
            spacing: 5
            Label { text: "Sensitivity"; color: "#87919d"; font.pixelSize: 10 }
            Label {
                Layout.fillWidth: true
                text: root.sensitivitySummary()
                color: "#c3cad3"
                font.pixelSize: 12
                wrapMode: Text.WordWrap
            }
            Label {
                Layout.fillWidth: true
                text: "Counts are generated server-side after Unix peer UID ownership is resolved."
                color: "#6f7985"
                font.pixelSize: 9
                wrapMode: Text.WordWrap
            }
        }
    }

    GridLayout {
        Layout.fillWidth: true
        columns: 4
        columnSpacing: 9
        rowSpacing: 9
        Repeater {
            model: root.layers
            delegate: Rectangle {
                required property var modelData
                Layout.fillWidth: true
                implicitHeight: 78
                radius: 12
                color: "#15191f"
                border.color: "#282e37"
                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 12
                    RowLayout {
                        Layout.fillWidth: true
                        Label { text: modelData.id; color: "#8ca0bc"; font.pixelSize: 10; font.bold: true }
                        Item { Layout.fillWidth: true }
                        Label { text: String(root.layerCount(modelData.id)); color: "white"; font.pixelSize: 18; font.bold: true }
                    }
                    Label { text: modelData.label; color: "#8f99a5"; font.pixelSize: 10 }
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
        interval: 4000
        repeat: true
        running: root.visible
        triggeredOnStart: true
        onTriggered: root.refresh()
    }
}
