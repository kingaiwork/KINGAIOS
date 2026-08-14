function primaryPanel() {
    for (const id of panelIds) {
        const candidate = panelById(id);
        if (!candidate) continue;
        candidate.currentConfigGroup = ["General"];
        if (String(candidate.readConfig("kingaiManaged")) === "true") return candidate;
    }
    if (panelIds.length > 0) {
        const fallback = panelById(panelIds[0]);
        if (fallback) return fallback;
    }
    return new Panel;
}
function clearWidgets(panel) {
    const ids = panel.widgetIds;
    for (const id of ids) {
        const widget = panel.widgetById(id);
        if (widget) widget.remove();
    }
}
function addIfAvailable(panel, type) {
    if (knownWidgetTypes.includes(type)) panel.addWidget(type);
}
function markManaged(panel, experience) {
    panel.currentConfigGroup = ["General"];
    panel.writeConfig("kingaiManaged", "true");
    panel.writeConfig("kingaiExperience", experience);
}

const panel = primaryPanel();
clearWidgets(panel);
panel.location = "bottom";
panel.lengthMode = "fill";
panel.alignment = "center";
panel.height = Math.round(gridUnit * 2.4);
panel.hiding = "none";
markManaged(panel, "kingai-classic");

addIfAvailable(panel, "org.kde.plasma.kickoff");
addIfAvailable(panel, "org.kingai.agentcenter");
addIfAvailable(panel, "org.kde.plasma.icontasks");
addIfAvailable(panel, "org.kde.plasma.panelspacer");
addIfAvailable(panel, "org.kde.plasma.systemtray");
addIfAvailable(panel, "org.kde.plasma.digitalclock");
addIfAvailable(panel, "org.kde.plasma.showdesktop");
