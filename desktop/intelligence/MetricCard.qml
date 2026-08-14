import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

Rectangle {
    id: root
    property string label: ""
    property string value: "—"
    property string note: ""

    Layout.fillWidth: true
    implicitHeight: 112
    radius: 15
    color: "#181c22"
    border.color: "#272d36"

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 15
        spacing: 4
        Label { text: root.label; color: "#8994a2"; font.pixelSize: 11 }
        Item { Layout.fillHeight: true }
        Label {
            Layout.fillWidth: true
            text: root.value
            color: "white"
            font.pixelSize: root.value.length > 12 ? 16 : 25
            font.bold: true
            elide: Text.ElideRight
        }
        Label {
            Layout.fillWidth: true
            text: root.note
            color: "#747e8a"
            font.pixelSize: 10
            elide: Text.ElideRight
        }
    }
}
