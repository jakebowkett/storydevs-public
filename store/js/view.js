
/*
    // page doesn't affect context...?

    State object fields:
        kind:       [mode/page/modal?],
        view:       [home/account/talent/etc],
        subView:    [talent/forums/etc],
        query:      [?s=A0D0/?all=true],
        resource:   [G7xH1slih],
        editing:    [G7xH1slih],
        editorKind: [new/edit/reply],
        layout:     [search/search-editor/detail/etc],
*/

function emptyModal(modal) {
    const cc = Array.from(modal.children).slice(2);
    for (let i = cc.length-1; i >= 0; i--) {
        removeNode(cc[i]);
    }
}

function populateModal(modal, s) {
    for (const elem of strToElems(s)) {
        modal.appendChild(elem);
    }
}

function showConfirmModal(e, name, onload) {
    showModal(name, window[onload]);
}

function insertEmail(res) {
    const email = q(`.wdgt-btn[name="email"] > .text`).textContent.trim();
    return res.replace(/\{\{e\u00ad?mail\}\}/g, email);
}

function showModal(name, onLoad) {
    
    const container = q("#modal_container");
    const modal = q("#modal");
    emptyModal(modal);
    modal.appendChild(strToElem(loading));
    container.classList.add("visible");
    
    window.addEventListener("keydown", escapeModal);
    context.modalVisible = true;
    
    get("/"+name+"/partial", function(err, res) {

        if (err) {
            log(err);
            return;
        }
        
        if (onLoad) {
            res = onLoad(res);
        }
        
        let temp = modal.cloneNode(true);
        emptyModal(temp);
        temp.id = "";
        temp.style.transitionDuration = "0s";
        temp.style.position = "absolute";
        temp.style.opacity = 0;
        temp.style.pointerEvents = "none";
        temp.style.height = "auto";
        
        populateModal(temp, res);
        container.appendChild(temp);
        
        modal.style.height = temp.clientHeight + "px";
        setTimeout(function() {
            emptyModal(modal);
            populateModal(modal, res);
            container.removeChild(temp);
            modal.style.height = "auto";
            init(modal);
            const input = q("input", modal)
            if (input) {
                input.focus();
            }
        }, getTransitionDuration(modal));
    });
}

function escapeModal(e) {
    if (e.key === "Escape") {
        dismissModal();
    }
}

function dismissModalBackdrop(e) {
    if (findAncestor("#modal", e.target)) {
        return;
    }
    dismissModal();
}

function dismissModal() {
    
    window.removeEventListener("keydown", escapeModal);
    context.modalVisible = false;
    
    if (context.modalInitially) {
        if (context.viewType === "page") {
            showPage(context.view);
        } else {
            showMode(context.view);
        }
        context.modalInitially = false;
    }
    
    let container = q("#modal_container");
    container.classList.remove("visible");
    
    setTimeout(function() {
        const modal = q("#modal");
        emptyModal(modal);
        modal.style.height = "";
    }, getTransitionDuration(container));
}

var layoutMapping = {
    "page": ["page"],
    "search-detail": ["search", "detail"],
    "search-editor": ["search", "editor"],
    "search": ["search", "browse"],
    "browse": ["search", "browse"],
    "detail": ["browse", "detail"],
    "editor": ["detail", "editor"],
};

function numOfCols() {
    const pseudo = cssPseudo(document.body, ":after", "content");
    return parseInt(pseudo.slice(1, -1));
}

function layoutEvent(e) {
    
    if (e.currentTarget.classList.contains("disabled")) {
        return;
    }
    
    const colNum = numOfCols();
    let target = e.currentTarget.dataset.col;
    
    if (colNum === 1) {
        setLayout(target, context.view, context.subView);
        return;
    }
    
    if (contains(layoutMapping[context.layout], target)) {
        return;
    }
    
    if (target === "browse" && contains(layoutMapping[context.layout], "editor")) {
        target = "detail";
    }
    
    setLayout(target, context.view, context.subView);
}

function setLayout(layout, view, subView, options) {
    let b = q("body");
    const layouts = [...keys(layoutMapping), "page"];
    const views = [...keys(context.page), ...keys(context.mode), "error"];
    
    /*
        This must be done prior to altering the
        body element classList as it uses those
        classes to figure out which columns will
        be moving.
    */
    if (!(options && options.suppressSetMoving)) {
        setMovingColumns(layout);
    }
    
    views.push("search-detail", "search-editor");
    for (const cls of layouts.concat(views)) {
        b.classList.remove(cls);
    }
    b.classList.add(layout);
    b.classList.add(view);
    if (subView) {
        b.classList.add(subView);
    }
    context.layout = layout;
}

function setMovingColumns(layout) {
    const cls = q("body").classList;
    const moving = [];
    let oldMapping;
    for (let i = 0; i < cls.length; i++) {
        const c = cls[i];
        oldMapping = layoutMapping[c];
        if (oldMapping !== undefined) {
            break;
        }
    }
    const newMapping = layoutMapping[layout];
    if (oldMapping[0] !== newMapping[0]) {
        moving.push(newMapping[0]);
    }
    /*
        The "page" mapping only has one column
        so its second index will be undefined.
    */
    if (oldMapping[1] !== newMapping[1] && newMapping[1] !== undefined) {
        moving.push(newMapping[1]);
    }
    for (const name of moving) {
        const col = q("#"+name);
        col.dataset.moving = "true";
        col.addEventListener("transitionend", colTransitionEnd);
    }
}

function colTransitionEnd(e) {
    const col = e.target;
    delete col.dataset.moving;
    col.removeEventListener("transitionend", colTransitionEnd);
    col.dispatchEvent(new Event("canload"));
}

function selectNavLink(view) {
    
    const links = qAll(`#sidebar [data-action="navLink"]`);

    for (let i = 0; i < links.length; i++) {
        
        let link = links[i];
        let href = link.getAttribute("href").slice(1);
        
        const ps = findAncestor("#persona_switcher", link);
        if (href == "account" && ps) {
            link = ps;
        }
        
        if (view in context.modal) {
            continue;
        }
        
        if (view === href) {
            link.classList.add("selected");
            continue;
        }
        link.classList.remove("selected");
    }
}

function deselectNavLinks() {
    let links = qAll(".link", q("#header"));
    for (let i = 0; i < links.length; i++) {
        let link = links[i];
        link.classList.remove("selected");
    }
}

function showMode(name) {
    
    q("body").dispatchEvent(new Event("viewchange"));
    
    document.title = context.mode[name].title;
    history.pushState({
        kind:   "mode",
        view:   name,
        layout: "search",
    }, "", "/"+name);
    selectNavLink(name);
    context.view = name;
    context.subView = null;
    context.query = null;
    context.resource = null;
    context.editing = null;
    
    const browseText = q('#sidebar .btn[data-col="browse"] .text');
    browseText.textContent = context.mode[name].browseName;
    setResourceKind(name);
    
    setLayout("search", name);
    setEmpty(name, ["browse", "detail", "editor"]);
    loadColumn("search", "/"+name);
}

function setResourceKind(mode) {
    const kind = context.mode[mode].resourceColumn;
    q('#sidebar .btn[data-col="detail"] .text').textContent = kind;
}

// No-op if cols.length === 0.
function setEmpty(mode, cols) {
    for (let name of cols) {
        let msg = context.empty[name].replace(/\n/g, "<br>");
        const col = q("#"+name);
        q(".col_inner", col).innerHTML = ('<div class="empty">' + msg + '</div>');
        updateScroll(q(".scroll", col), true, true);
    }
}

function loadColumn(col, path, query, anchor) {
    
    const column = q("#"+col);
    const inner = q(".col_inner", column);
    inner.innerHTML = loading;
    updateScroll(q("#"+col+" > .scroll"), false, true);
    
    path += "/partial"
    
    if (query) {
        path += query;
    }
    
    get(path, function(err, res) {
        
        if (err) {
            throw "err requesting partial mode";
            return;
        }
        if (!res) {
            throw "no res object";
            return;
        }
        
        if (column.dataset.moving) {
            column.addEventListener("canload", load);
        } else {
            load();
        }
        
        function load(e) {
            inner.innerHTML = res.col;
            column.removeEventListener("canload", load);
            init(column);
            if (anchor) {
                scrollTo(q(anchor), {padding: scrollToPad});
            }
        }
    });
}

function search(e) {
    
    e.preventDefault();
    
    const form = findAncestor("form", e.target);
    const pf = parseForm(form, true);
    if (!pf) {
        return;
    }
    
    const path = "/"+context.view;
    let json = pf.get("json");
    json = JSON.parse(json);
    const s = compactQuery(json);
    
    let query = "?";
    if (s !== "") {
        query += "s=" + s;
    }
    if (query === "?") {
        query += "all=true";
    }
    
    setLayout("browse", context.view);
    context.query = query;
    history.pushState({
        kind:   "mode",
        view:   context.view,
        query:  query,
        layout: "browse",
    }, "", path+query);
    loadColumn("browse", path, query);
}

function compactQuery(json) {
    let s = "";
    for (const k in json) {
        let v = json[k];
        if (Array.isArray(v)) {
            if (typeof v[0] === "object") {
                s += k + "0"; // subgroups need a dummy value
                for (let i = 0; i < v.length; i++) {
                    s += compactQuery(v[i]);
                    if (i !== v.length-1) {
                        s += k + "1"; // continuing value
                    }
                }
                s += k + "2"; // terminating value
                continue;
            }
            v = v.join(".");
        } else if (typeof v === "object") {
            s += k + "0"; // subgroups need a dummy value
            s += compactQuery(v);
            s += k + "2"; // terminating value
            continue;
        }
        s += k + v;
    }
    return s;
}

function resource(e) {
    
    e.preventDefault();
    
    let results = qAll(".result", e.currentTarget.parentNode);
    for (let i = 0; i < results.length; i++) {
        let result = results[i];
        if (e.currentTarget == result) {
            result.classList.add("selected");
            continue;
        }
        result.classList.remove("selected");
    }
    
    const href = e.currentTarget.getAttribute("href");
    const parts = urlToParts(href);
    context.editing = null;
    if (context.subView) {
        if (context.subView === "settings") {
            context.editing = parts.pathSeg[2];
        }
        context.resource = parts.pathSeg[2];
    } else {
        context.resource = parts.pathSeg[1];
    }
    
    history.pushState({
        kind:     "mode",
        view:     context.view,
        subView:  context.subView,
        resource: context.resource,
        layout:   "detail",
    }, "", href);

    setEmpty(context.view, ["editor"]);
    
    setLayout("detail", context.view, context.subView);
    loadColumn("detail", parts.path, null, parts.anchor);
}

function replyResource(e) {
    
    e.preventDefault();
    
    const c = context;
    let route;
    if (c.subView) {
        route = "/"+c.view+"/"+c.subView+"/"+c.resource+"/reply";
    } else {
        route = "/"+c.view+"/"+c.resource+"/reply";
    }
    
    setLayout("editor", c.view, c.subView);
    history.pushState({
        kind:       "mode",
        view:       c.view,
        subView:    c.subView,
        resource:   c.resource,
        editorKind: "reply",
        layout:     "editor",
    }, "", route);
    loadColumn("editor", route);
}

function newResource(e) {
    
    e.preventDefault();
    const c = context;
    
    if (!c.loggedIn) {
        showNotification("Unauthorised", "You must be logged in to perform this action.", "warn");
        return;
    }
    
    let route;
    if (c.subView) {
        route = "/"+c.view+"/"+c.subView+"/new";
    } else {
        route = "/"+c.view+"/new";
    }
    
    setLayout("search-editor", c.view, c.subView);
    history.pushState({
        kind:      "mode",
        view:       c.view,
        subView:    c.subView,
        editorKind: "new",
        layout:     "search-editor",
    }, "", route);
    loadColumn("editor", route);
}

function editResource(e) {
    
    e.preventDefault();
    
    // Get resource slug.
    const parts = e.currentTarget.getAttribute("href").split("/");
    const slug = parts[parts.length-2];

    const c = context;
    let route;
    if (c.subView) {
        route = "/"+c.view+"/"+c.subView+"/"+slug+"/edit";
    } else {
        route = "/"+c.view+"/"+slug+"/edit";
    }
    
    setLayout("editor", c.view, c.subView);
    history.pushState({
        kind:        "mode",
        view:        c.view,
        subView:     c.subView,
        resource:    c.resource,
        editorKind:  "edit",
        editing:     slug,
        layout:      "editor",
    }, "", route);
    context.editing = slug;
    
    loadColumn("editor", route);
}

function deleteAccount(e) {
    clientSideLogout();
    del("/delete_account", (err, res) => {
        if (err) {
            log(err);
            showNotification(
                "Error",
                "Unable to delete your account at this time.",
                "error"
            );
            return;
        }
        showNotification(
            "Success!",
            "Your account has successfully been deleted.",
            "success"
        );
    });
}
    
function deleteResourcePrompt(e, nameOverride) { 
    
    e.preventDefault();
    const c = context;
    const view = c.subView ? c.subView : c.view;
    
    // Get resource slug.
    const parts = e.currentTarget.getAttribute("href").split("/");
    const slug = parts[parts.length-2];
    
    let resourceName = c.mode[view].resourceName;
    if (nameOverride) {
        resourceName = nameOverride;
    }
    
    showModal("delete", function(s) {
        s = s.replace(/deleteResource/, `deleteResource[${slug}]`);
        s = s.replace(/\[re\u00ad?source\]/, resourceName);
        return s.replace(/\[re\u00ad?source\]/, resourceName.toLowerCase());
    });
}

function deleteResource(e, slug) {
    
    e.preventDefault();
    
    const c = context;
    let resourceName;
    let route;
    let meta;
    if (c.subView) {
        route = `/${c.view}/${c.subView}/${slug}`;
        resourceName = c.mode[c.subView].resourceName;
        meta = c.subView;
    } else {
        route = `/${c.view}/${slug}`;
        resourceName = c.mode[c.view].resourceName;
        meta = c.view;
    }
    
    del(route, (err, res) => {
        if (err) {
            showNotification(
                "Error",
                "Unable to delete your " + resourceName + " at this time.",
                "error"
            );
            dismissModal();
            return;
        }
        // User error.
        if (res && res.feedback) {
            const form = q("#modal form");
            addFeedback(res.feedback, form);
            updateModalHeight(form);
            return;
        }
        dismissModal();
        showNotification(
            "Success!",
            "Your " + resourceName + " has successfully been deleted.",
            "success"
        );
        setLayout("search", c.view, c.subView);
        history.pushState({
            kind:    "mode",
            view:    c.view,
            subView: c.subView,
            query:   c.query,
            layout:  "search",
        }, "", "/"+c.view);
        c.resource = null;
        c.editing = null;
        setEmpty(meta, ["detail", "editor"]);
        const result = q(`#browse a[data-slug="${slug}"]`);
        if (result) {
            removeNode(result);
        }
    });
}

function showPage(name, options) {
    
    q("body").dispatchEvent(new Event("viewchange"));
    
    const page = q("#page");
    const pageBody = q("#page_body");
    
    document.title = context.page[name].title;
    setLayout("page", name);
    
    const path = "/"+name+"/partial";
    if (name === "home") {
        name = "";
    }
    
    if (!(options && options.suppressHistory)) {
        history.pushState({
            kind:   "page",
            view:   name,
            layout: "page",
        }, "", "/"+name);
    }
    selectNavLink(name);
    
    pageBody.innerHTML = loading;
    updateScroll(page, false, true);
    
    get(path, function(err, res) {
        
        if (err) {
            log("err requesting partial page");
            return
        }
        
        if (!res) {
            log("no res object");
            return
        }
        
        
        if (page.dataset.moving) {
            page.addEventListener("canload", load);
        } else {
            load();
        }
        
        function load(e) {
            pageBody.innerHTML = res.page;
            page.removeEventListener("canload", load);
            initActionables(pageBody);
            updateScroll(page, true, true);
        }
    });
}

function showError(name, err) {
    
    if (!err) {
        throw "no error to display";
        return;
    }
 
    const page = q("#page");
    const pageBody = q("#page_body");
    
    document.title = context.page[name].title;
    setLayout("page", "error");
    deselectNavLinks();
  
    pageBody.innerHTML = err;
    updateScroll(page, true, true);
}

function setNumOfCols(single) {
    
    const body = document.body;
    const btns = qAll("#sidebar .layout .btn");
    
    if (single) {
        body.classList.add("single");
        body.classList.remove("dual");
        btns[0].classList.add("selected");
        btns[1].classList.remove("selected");
    } else {
        body.classList.remove("single");
        body.classList.add("dual");
        btns[0].classList.remove("selected");
        btns[1].classList.add("selected");
    }
    
    let n = getTransitionDuration(q(".col"));
    setTimeout(function(){
        updateScrolls();
    }, n);
}
