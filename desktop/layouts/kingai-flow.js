function primaryPanel() {
    let panel = null;
    if (panelIds.length > 0) panel = panelById(panelIds[0]);
    if (!panel) panel = new Panel;
    return panel;
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
panel.lengthMode = "fit";
panel.alignment = "center";
panel.height = Math.round(gridUnit * 3.0);
panel.hiding = "dodgewindows";
markManaged(panel, "kingai-flow");

addIfAvailable(panel, "org.kde.plasma.kickoff");
addIfAvailable(panel, "org.kingai.agentcenter");
addIfAvailable(panel, "org.kde.plasma.pager");
addIfAvailable(panel, "org.kde.plasma.icontasks");
addIfAvailable(panel, "org.kde.plasma.systemtray");
addIfAvailable(panel, "org.kde.plasma.digitalclock");
