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
panel.location = "bottom";
panel.lengthMode = "fit";
panel.alignment = "center";
panel.height = Math.round(gridUnit * 3.0);
// Flow is the calm dock-oriented experience. Keep its primary controls visible
// by default; users can opt into auto-hide later from Plasma settings.
panel.hiding = "none";
addIfAvailable(panel, "org.kde.plasma.kickoff");
addIfAvailable(panel, "org.kingai.agentcenter");
addIfAvailable(panel, "org.kde.plasma.icontasks");
addIfAvailable(panel, "org.kde.plasma.systemtray");
addIfAvailable(panel, "org.kde.plasma.digitalclock");
panel.reloadConfig();

const wallpaper = "file:///usr/share/kingai/desktop/wallpapers/kingai-flow.jpg";
for (const desktop of desktops()) {
    desktop.wallpaperPlugin = "org.kde.image";
    desktop.currentConfigGroup = ["Wallpaper", "org.kde.image", "General"];
    desktop.writeConfig("Image", wallpaper);
    desktop.reloadConfig();
}
