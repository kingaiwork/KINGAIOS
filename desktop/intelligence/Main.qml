import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import QtCore

ApplicationWindow {
    id: root
    width: 1280
    height: 820
    minimumWidth: 1040
    minimumHeight: 680
    visible: true
    title: "KINGAI Intelligence"
    color: "#101216"

    property string selectedCenter: "home"
    property string runtimeHealth: "offline"
    property string version: "—"
    property string policyMode: "—"
    property int registeredAgents: 0
    property int activeTasks: 0
    property int runningTasks: 0
    property int waitingTasks: 0
    property int waitingApprovalTasks: 0
    property int blockedTasks: 0
    property int pausedTasks: 0
    property int planningTasks: 0
    property int pendingApprovals: 0
    property int modelProviders: 0
    property string modelMode: "—"
    property string modelStrategy: "—"
    property string memoryMode: "—"
    property bool cloudRequired: false
    property string updatedAt: "—"

    readonly property var centers: [
        { id: "home", label: "Home", glyph: "⌂", title: "Intelligence Home", note: "Your governed local AI workspace.", detail: "See runtime health, active work and the system surfaces that need your attention." },
        { id: "agents", label: "Agents", glyph: "A", title: "Agent Center", note: "Identity before authority.", detail: "KINGAI agents operate through named identities and capability policy. Privileged roles remain separated from ordinary agents." },
        { id: "tasks", label: "Tasks", glyph: "T", title: "Task Center", note: "Goals become governed task graphs.", detail: "See safe lifecycle counts for local tasks without exposing task goals, targets or results through the public desktop channel." },
        { id: "approvals", label: "Approvals", glyph: "✓", title: "Approval Center", note: "Human authority stays explicit.", detail: "High-risk capabilities require scoped, expiring decisions. The overview exposes only pending counts; decisions remain behind the governed runtime path." },
        { id: "memory", label: "Memory", glyph: "M", title: "Memory Center", note: "Local-first intelligent state.", detail: "Memory is owned, sensitivity-aware and governed. Content is never exposed through the public desktop status channel." },
        { id: "models", label: "Models", glyph: "◈", title: "Model Center", note: "One fabric across local and remote providers.", detail: "Model status reflects configured provider count and routing mode while private credentials remain outside the desktop overview." },
        { id: "automations", label: "Automations", glyph: "↻", title: "Automation Center", note: "Repeatable intelligence with visible control.", detail: "Automation work shares the same Task, Policy, Approval and Audit contracts. Current overview uses task lifecycle state as its safe activity signal." },
        { id: "health", label: "System Health", glyph: "+", title: "System Health", note: "Understand before repairing.", detail: "Health shows sanitized runtime, policy and dependency posture. Repair and privileged actions remain governed separately." }
    ]

    function centerById(id) {
        for (var i = 0; i < centers.length; ++i) if (centers[i].id === id) return centers[i]
        return centers[0]
    }

    function normalizeCenter(value) {
        var v = value || ""
        if (v.indexOf("kingai://") === 0) {
            v = v.substring(9)
            if (v.indexOf("/") >= 0) v = v.split("/")[0]
        }
        for (var i = 0; i < centers.length; ++i) if (centers[i].id === v) return v
        return "home"
    }

    function parseArguments() {
        var args = Application.arguments || []
        for (var i = 0; i < args.length; ++i) {
            if (args[i] === "--center" && i + 1 < args.length) {
                selectedCenter = normalizeCenter(args[i + 1])
                return
            }
            if (String(args[i]).indexOf("kingai://") === 0) {
                selectedCenter = normalizeCenter(args[i])
                return
            }
        }
    }

    function metricsForCenter(center) {
        switch (center) {
        case "tasks":
            return [
                { label: "Active", value: String(activeTasks), note: "non-terminal" },
                { label: "Running", value: String(runningTasks), note: "executing now" },
                { label: "Waiting", value: String(waitingTasks + waitingApprovalTasks), note: waitingApprovalTasks > 0 ? String(waitingApprovalTasks) + " need approval" : "queued / dependent" },
                { label: "Blocked", value: String(blockedTasks), note: blockedTasks > 0 ? "needs attention" : "clear" }
            ]
        case "approvals":
            return [
                { label: "Pending", value: String(pendingApprovals), note: pendingApprovals > 0 ? "owner decision needed" : "clear" },
                { label: "Tasks waiting", value: String(waitingApprovalTasks), note: "waiting approval" },
                { label: "Running", value: String(runningTasks), note: "already authorized" },
                { label: "Policy", value: policyMode, note: "governed" }
            ]
        case "memory":
            return [
                { label: "Mode", value: memoryMode, note: "local memory posture" },
                { label: "Content", value: "Private", note: "not published here" },
                { label: "Cloud required", value: cloudRequired ? "Yes" : "No", note: cloudRequired ? "dependency present" : "local-first" },
                { label: "Runtime", value: runtimeHealth === "ok" ? "Healthy" : "Offline", note: version }
            ]
        case "models":
            return [
                { label: "Providers", value: modelProviders > 0 ? String(modelProviders) : "—", note: modelProviders > 0 ? "configured" : "not configured" },
                { label: "Mode", value: modelMode, note: "routing default" },
                { label: "Strategy", value: modelStrategy, note: "provider policy" },
                { label: "Cloud required", value: cloudRequired ? "Yes" : "No", note: cloudRequired ? "dependency present" : "offline-capable core" }
            ]
        case "automations":
            return [
                { label: "Active work", value: String(activeTasks), note: "task-backed activity" },
                { label: "Planning", value: String(planningTasks), note: "preparing execution" },
                { label: "Paused", value: String(pausedTasks), note: "held by user/system" },
                { label: "Blocked", value: String(blockedTasks), note: blockedTasks > 0 ? "needs attention" : "clear" }
            ]
        case "health":
            return [
                { label: "Runtime", value: runtimeHealth === "ok" ? "Healthy" : "Offline", note: version },
                { label: "Policy", value: policyMode, note: "authority control" },
                { label: "Blocked tasks", value: String(blockedTasks), note: blockedTasks > 0 ? "investigate" : "clear" },
                { label: "Cloud required", value: cloudRequired ? "Yes" : "No", note: cloudRequired ? "external dependency" : "local survival path" }
            ]
        case "agents":
            return [
                { label: "Agents", value: String(registeredAgents), note: "registered identities" },
                { label: "Active tasks", value: String(activeTasks), note: "shared workload" },
                { label: "Approvals", value: String(pendingApprovals), note: pendingApprovals > 0 ? "pending authority" : "clear" },
                { label: "Policy", value: policyMode, note: "identity-bound" }
            ]
        default:
            return [
                { label: "Agents", value: String(registeredAgents), note: "registered" },
                { label: "Tasks", value: String(activeTasks), note: "active" },
                { label: "Approvals", value: String(pendingApprovals), note: pendingApprovals > 0 ? "need attention" : "clear" },
                { label: "Models", value: modelProviders > 0 ? String(modelProviders) : "—", note: modelProviders > 0 ? modelMode : "not configured" }
            ]
        }
    }

    function stateSummary() {
        if (selectedCenter === "tasks" || selectedCenter === "automations") {
            return "Running " + runningTasks + "  ·  Planning " + planningTasks + "  ·  Waiting " + waitingTasks + "  ·  Approval " + waitingApprovalTasks + "  ·  Paused " + pausedTasks + "  ·  Blocked " + blockedTasks
        }
        if (selectedCenter === "models") {
            return modelProviders > 0 ? (modelProviders + " provider(s) · " + modelMode + " · " + modelStrategy) : "No model providers configured yet. The governed OS core remains available without publishing provider credentials."
        }
        if (selectedCenter === "approvals") {
            return pendingApprovals > 0 ? (pendingApprovals + " approval request(s) are pending. Open a governed approval client to inspect scope and decide.") : "No pending approval requests in the sanitized status snapshot."
        }
        if (selectedCenter === "health") {
            return "Runtime " + runtimeHealth + " · Policy " + policyMode + " · Cloud required " + (cloudRequired ? "yes" : "no")
        }
        return "Memory " + memoryMode + " · Models " + modelMode + " · Policy " + policyMode
    }

    Component.onCompleted: parseArguments()

    Rectangle {
        anchors.fill: parent
        color: "#101216"

        RowLayout {
            anchors.fill: parent
            spacing: 0

            Rectangle {
                Layout.preferredWidth: 232
                Layout.fillHeight: true
                color: "#15181d"

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 18
                    spacing: 10

                    Label { text: "KINGAI"; color: "white"; font.pixelSize: 23; font.bold: true }
                    Label { text: "INTELLIGENCE"; color: "#8793a3"; font.pixelSize: 10; font.letterSpacing: 2 }
                    Item { Layout.preferredHeight: 10 }

                    Repeater {
                        model: root.centers
                        delegate: Button {
                            required property var modelData
                            Layout.fillWidth: true
                            implicitHeight: 44
                            text: modelData.glyph + "   " + modelData.label
                            checked: root.selectedCenter === modelData.id
                            checkable: true
                            autoExclusive: true
                            onClicked: root.selectedCenter = modelData.id
                        }
                    }

                    Item { Layout.fillHeight: true }
                    Label {
                        text: runtimeHealth === "ok" ? "● Runtime online" : "● Runtime unavailable"
                        color: runtimeHealth === "ok" ? "#72d58a" : "#e58a85"
                        font.pixelSize: 11
                    }
                    Label { text: version; color: "#6f7885"; font.pixelSize: 10 }
                }
            }

            ScrollView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                clip: true

                ColumnLayout {
                    width: Math.max(760, root.width - 232)
                    spacing: 18

                    Item { Layout.preferredHeight: 24 }

                    RowLayout {
                        Layout.fillWidth: true
                        Layout.leftMargin: 34
                        Layout.rightMargin: 34

                        ColumnLayout {
                            Layout.fillWidth: true
                            spacing: 4
                            Label { text: root.centerById(root.selectedCenter).title; color: "white"; font.pixelSize: 30; font.bold: true }
                            Label { text: root.centerById(root.selectedCenter).note; color: "#9da7b4"; font.pixelSize: 14 }
                        }
                        Rectangle {
                            implicitWidth: 112
                            implicitHeight: 34
                            radius: 17
                            color: runtimeHealth === "ok" ? "#17351f" : "#3a1f20"
                            Label {
                                anchors.centerIn: parent
                                text: runtimeHealth === "ok" ? "●  ONLINE" : "●  OFFLINE"
                                color: runtimeHealth === "ok" ? "#9ee7ad" : "#f0aaa6"
                                font.pixelSize: 10
                                font.bold: true
                            }
                        }
                    }

                    Label {
                        Layout.fillWidth: true
                        Layout.leftMargin: 34
                        Layout.rightMargin: 34
                        text: root.centerById(root.selectedCenter).detail
                        color: "#c1c7cf"
                        wrapMode: Text.WordWrap
                        font.pixelSize: 14
                    }

                    GridLayout {
                        Layout.fillWidth: true
                        Layout.leftMargin: 34
                        Layout.rightMargin: 34
                        columns: 4
                        columnSpacing: 12
                        rowSpacing: 12

                        Repeater {
                            model: root.metricsForCenter(root.selectedCenter)
                            delegate: Rectangle {
                                required property var modelData
                                Layout.fillWidth: true
                                implicitHeight: 112
                                radius: 15
                                color: "#181c22"
                                border.color: "#272d36"
                                ColumnLayout {
                                    anchors.fill: parent
                                    anchors.margins: 15
                                    Label { text: modelData.label; color: "#8994a2"; font.pixelSize: 11 }
                                    Item { Layout.fillHeight: true }
                                    Label { text: modelData.value; color: "white"; font.pixelSize: modelData.value.length > 12 ? 16 : 25; font.bold: true; elide: Text.ElideRight; Layout.fillWidth: true }
                                    Label { text: modelData.note; color: "#747e8a"; font.pixelSize: 10; elide: Text.ElideRight; Layout.fillWidth: true }
                                }
                            }
                        }
                    }

                    Rectangle {
                        Layout.fillWidth: true
                        Layout.leftMargin: 34
                        Layout.rightMargin: 34
                        implicitHeight: 250
                        radius: 18
                        color: "#161a20"
                        border.color: "#272d36"

                        ColumnLayout {
                            anchors.fill: parent
                            anchors.margins: 22
                            spacing: 12

                            Label {
                                text: root.selectedCenter === "home" ? "One Desktop. One governed intelligence core." : root.centerById(root.selectedCenter).title
                                color: "white"
                                font.pixelSize: 19
                                font.bold: true
                            }

                            Label {
                                Layout.fillWidth: true
                                text: root.selectedCenter === "home"
                                      ? "KINGAI Intelligence is the flagship AI-first experience inside KINGAI OS Desktop. Flow and Classic change interaction style; Policy, Approval, Task, Memory, Model and Audit remain the same trusted operating-system core."
                                      : root.centerById(root.selectedCenter).detail
                                color: "#abb4bf"
                                wrapMode: Text.WordWrap
                                font.pixelSize: 13
                            }

                            Rectangle {
                                Layout.fillWidth: true
                                implicitHeight: 62
                                radius: 12
                                color: "#111419"
                                RowLayout {
                                    anchors.fill: parent
                                    anchors.margins: 14
                                    ColumnLayout {
                                        Layout.fillWidth: true
                                        Label { text: "Safe status summary"; color: "#7f8995"; font.pixelSize: 10 }
                                        Label { text: root.stateSummary(); color: "white"; font.pixelSize: 12; elide: Text.ElideRight; Layout.fillWidth: true }
                                    }
                                    ColumnLayout {
                                        Layout.preferredWidth: 230
                                        Label { text: "Last status update"; color: "#7f8995"; font.pixelSize: 10 }
                                        Label { text: root.updatedAt; color: "white"; font.pixelSize: 11; elide: Text.ElideRight; Layout.fillWidth: true }
                                    }
                                }
                            }

                            Label {
                                Layout.fillWidth: true
                                text: "Privacy boundary: this desktop shell reads aggregate status only. Prompt text, task goals, approval targets, secrets, credentials and memory content are never read from the public status channel."
                                color: "#78828e"
                                wrapMode: Text.WordWrap
                                font.pixelSize: 10
                            }
                        }
                    }

                    Item { Layout.preferredHeight: 34 }
                }
            }
        }
    }

    function refreshStatus() {
        var xhr = new XMLHttpRequest()
        xhr.onreadystatechange = function() {
            if (xhr.readyState !== XMLHttpRequest.DONE) return
            if (xhr.status !== 0 && xhr.status !== 200) {
                root.runtimeHealth = "offline"
                return
            }
            try {
                var s = JSON.parse(xhr.responseText)
                root.runtimeHealth = s.health || "offline"
                root.version = s.version || "—"
                root.policyMode = s.policy || "—"
                root.registeredAgents = s.registered_agents || 0
                root.activeTasks = s.active_tasks || 0
                root.runningTasks = s.running_tasks || 0
                root.waitingTasks = s.waiting_tasks || 0
                root.waitingApprovalTasks = s.waiting_approval_tasks || 0
                root.blockedTasks = s.blocked_tasks || 0
                root.pausedTasks = s.paused_tasks || 0
                root.planningTasks = s.planning_tasks || 0
                root.pendingApprovals = s.pending_approvals || 0
                root.modelProviders = s.model_providers || 0
                root.modelMode = s.model_mode || "—"
                root.modelStrategy = s.model_strategy || "—"
                root.memoryMode = s.memory_mode || "—"
                root.cloudRequired = s.cloud_required === true
                root.updatedAt = s.updated_at || "—"
            } catch (e) {
                root.runtimeHealth = "offline"
            }
        }
        xhr.open("GET", "file:///run/kingai/public-status.json")
        xhr.send()
    }

    Timer {
        interval: 4000
        repeat: true
        running: true
        triggeredOnStart: true
        onTriggered: root.refreshStatus()
    }
}
