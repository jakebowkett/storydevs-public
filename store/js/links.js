
function localLink(e) {
    
    e.preventDefault();
    
    const link = e.currentTarget;
    const path = trim(link.getAttribute("href"), "/");
    const segs = path.split("/");
    const view = segs[0];
    
    if (view in context.modal) {
        showModal(view);
        return;
    }
    if (view in context.page) {
        showPage(view);
        return;
    }
    
    // Assume mode from here onward.
    selectNavLink(view);
    
    const hasSubView = context.mode[view].hideTools;
    let subView, resource;
    if (hasSubView) {
        if (segs.length >= 2) {
            subView = segs[1];
        }
        if (segs.length >= 3) {
            resource = segs[2];
        }
    } else if (segs.length >= 2) {
        resource = segs[1];
    }
    
    const toLoad = [];
    if (resource) {
        context.resource = resource;
        toLoad.push({col: "detail", route: link});
        setNumOfCols(true);
    }
    if (subView) {
        context.subView = subView;
        toLoad.push({col: "browse", route: `/${view}/${subView}`});
    }
    context.view = view;
    toLoad.push({col: "search", route: "/"+view});
    
    setResourceKind(view);
    history.pushState({}, "", toLoad[0].route);
    setLayout(toLoad[0].col, view, subView);
    for (const tl of toLoad) {
        loadColumn(tl.col, tl.route);
    }
}