function primaryPanel() {
    let panel = null;
    if (panelIds.length > 0) panel = panelById(panelIds[0]);
    if (!panel) panel = new Panel;
    return panel;
}
function hasWidget(panel, type) {
    for (const id of panel.widgetIds) {
        const w = panel.widgetById(id);
        if (w && w.type === type) return true;
    }
    return false;
}
function addIfAvailable(panel, type) {
    if (knownWidgetTypes.includes(type) && !hasWidget(panel, type)) panel.addWidget(type);
}
const panel = primaryPanel();
panel.location = "left";
panel.lengthMode = "fill";
panel.alignment = "center";
panel.height = Math.round(gridUnit * 3.1);
panel.hiding = "dodgewindows";
addIfAvailable(panel, "org.kde.plasma.kickoff");
addIfAvailable(panel, "org.kingai.agentcenter");
addIfAvailable(panel, "org.kde.plasma.icontasks");
addIfAvailable(panel, "org.kde.plasma.systemtray");
addIfAvailable(panel, "org.kde.plasma.digitalclock");
