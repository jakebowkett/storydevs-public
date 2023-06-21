
function menuClick(e) {
    
    const item = findAncestor(".btn", e.target);
    if (!item) {
        return;
    }
    
    e.preventDefault();
    
    if (item.hasAttribute("href")) {
        loadMenuItem(item);
    } else {
        toggleSubMenu(item);
    }
}

function toggleSubMenu(item) {
    
    if (item.classList.contains("expanded")) {
        item.classList.remove("expanded");
    } else {
        const menu = findAncestor(".wdgt-menu", item);
        const items = qAll(".btn", menu);
        item.classList.add("expanded");
    }
    
    const n = getTransitionDuration(q(".sub > .btn"));
    setTimeout(function() {
        updateScroll(findAncestor(".scroll", item), true, true);
    }, n + 50);
}

function loadMenuItem(item) {
    
    const menu = findAncestor(".wdgt-menu", item);
    const items = qAll(".btn", menu);
    for (const item of items) {
        item.classList.remove("selected");
    }
    item.classList.add("selected");
    
    const route = item.getAttribute("href");
    const parts = route.slice(1).split("/");
    const submode = parts[1];
    const inner = q("#browse .col_inner");
    const path = item.getAttribute("href") + "/partial";
    
    setEmpty(submode, ["detail", "editor"]);

    inner.innerHTML = loading;
    setResourceKind(submode);
    
    const c = context
    document.title = c.mode[c.view].title + " | " + c.mode[submode].resourcePlural;
    c.subView = submode;
    history.pushState({
        kind:    "mode",
        view:    c.view,
        subView: c.subView,
        layout:  "browse",
    }, "", route);
    setLayout("browse", c.view, submode);

    get(path, function(err, res) {
        if (err) {
            throw err
            return;
        }
        inner.innerHTML = res.col;
        init(q("#browse"));
    });
}
