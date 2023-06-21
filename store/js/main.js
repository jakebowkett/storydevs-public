
var rem;
var loading;
var scrollToPad;
var networkDelay = 0; // seconds
var scrollGap = 100; // in pixels
var mobile = false;

function main() {

    mobile = cssPx(q("#mobile"), "width") === 1;
    if (!mobile) {
        document.body.classList.add("hover");
    }

    document.body.classList.add(numOfCols() === 1 ? "single" : "dual");

    if (context.modalVisible) {
        window.addEventListener("keydown", escapeModal);
        requestAnimationFrame(function() {
            q("input").focus();
        });
    }
            
    // TODO: These could be templates
    let loadElem = q("#loading");
    loading = loadElem.parentNode.removeChild(loadElem);
    loading.removeAttribute("id");
    loading = loading.outerHTML;
    
    rem = q("#rem").clientHeight;
    scrollToPad = rem * 4;
    init();
    scriptLoaded = true;

    scrollToPost();
    
    window.onresize = function() {
        dismissMobileMenu();
        updateScrolls();
        updateEditorToolsPos();
        updateCanvasPos();
        updateSelection();
        updateRanges();
    }
    window.addEventListener("copy", storydevsCopy);

    initHistoryState();
    
    let n = getTransitionDuration(q("#search"));
    
    window.matchMedia("(min-width: 76rem)").addListener(function(e) {
        
        let animating = true;
        
        setTimeout(function() {
            animating = false;
        }, n);
        
        function frame() {
            requestAnimationFrame(function() {
                updateScrolls();
                updateEditorToolsPos();
                updateCanvasPos();
                updateSelection();
                updateRanges();
                if (animating) {
                    frame();
                }
            });
        }
        
        frame();
    });
}

function initHistoryState() {
    const c = context;
    const loc = document.location;
    const url = loc.pathname + loc.search;
    history.replaceState({
        kind:       c.viewType,
        view:       c.view,
        subView:    c.subView,
        query:      c.query,
        resource:   c.resource,
        editing:    c.editing,
        editorKind: c.editorKind,
        layout:     c.layout,
    }, "", url);
    window.addEventListener("popstate", popStateHandler);
}

function popStateHandler(e) {
    
    const s = e.state;
    if (s.kind === "page") {
        showPage(s.view, {suppressHistory: true});
        return;
    }

    document.title = context.mode[s.view].title;
    selectNavLink(s.view);
    setResourceKind(s.view);

    const notSeen = {
        browse: true,
        detail: true,
        editor: true,
    };
    let path = "/"+s.view;
    loadColumn("search", path);
    if (s.query) {
        delete notSeen.browse;
        loadColumn("browse", path, s.query);
    }
    if (s.subView) {
        delete notSeen.browse;
        path += "/"+s.subView;
        loadColumn("browse", path);
    }
    if (s.resource) {
        delete notSeen.resource;
        loadColumn("detail", path+"/"+s.resource);
    }
    if (s.editing) {
        delete notSeen.editor;
        loadColumn("editor", path+"/"+s.editing+"/edit");
    }
    if (s.editorKind && s.editorKind !== "edit") {
        delete notSeen.editor;
        path += "/"+s.editorKind;
        loadColumn("editor", path);
    }
    setLayout(s.layout, s.view, s.subView);
    setEmpty(s.view, keys(notSeen));
}

function storydevsCopy(e) {
    if (focusedEditor) {
        editorCopy(e);
        return;
    }
    e.preventDefault();
    let text = window.getSelection()
        .toString()
        .replace(/\u{00AD}+/ug, '');
    e.clipboardData.setData('text/plain', text);
}

function init(elem) {   
    initActionables(elem);
    initScrolls(elem);
    initTooltips(elem);
}

function navLink(e) {
    
    e.preventDefault();
    
    let link = e.currentTarget;
    let view = link.getAttribute("href").slice(1);

    dismissMobileMenu();
    
    if (view === "") {
        view = "home";
    }
    if (view in context.modal) {
        showModal(view);
        return;
    }
    if (context.modalVisible) {
        context.modalInitially = false;
        dismissModal();
    }
    if (view in context.page) {
        showPage(view);
        return;
    }
    if (view in context.mode) {
        showMode(view);
        return;
    }
}

function logout(e) {
    e.preventDefault();
    del("/logout", function(err) {
        if (err) {
            log(err);
            showNotification(
                "Error",
                "Unable to log out at this time.",
                "error"
            );
            return;
        }
        clientSideLogout();
        dismissMobileMenu();
        showNotification(
            "Success!",
            "You've logged out.",
            "success"
        );
    });
}

// Basically all the actions we take on the client side to signal
// that the user has been logged out. This is called when logging
// out but also when deleting one's own account.
function clientSideLogout() {

    const c = context;
    c.loggedIn = false;
    q("#auth").innerHTML = ".logged_in { display: none !important; }";

    for (const elem of qAll(".logout_remove")) {
        removeNode(elem);
    }

    const body = q("body");
    for (const m in c.mode) {
        if (c.mode[m].logoutRemove) {
            body.classList.remove(m);
            delete c.mode[m];
        }
    }

    // If we're in the account mode or we're in
    // a view that's not a known mode or page.
    if (c.view === "account" || (!c.mode[c.view] && !c.page[c.view])) {
        setEmpty(c.subView, ["search", "browse", "detail", "editor"]);
        showPage("home");
        return;
    }

    // Return if we're not in a mode.
    if (!c.mode[c.view]) {
        return;
    }

    // If we are, ensure the editor is emptied
    // and navigated away from.
    setEmpty(c.view, ["editor"]);
    let layout;
    for (const k in layoutMapping) {
        if (body.classList.contains(k)) {
            layout = k;
            break;
        }
    }
    if (layout === "editor") {
        setLayout("detail", c.view, c.subView);
        return;
    }
    if (layout === "search-editor") {
        setLayout("search-detail", c.view, c.subView);
    }
}

function toggleMobileMenu() {
    const menu = q("#mobile_menu");
    const backdrop = q("#mobile_backdrop");
    if (menu.classList.contains("expanded")) {
        menu.classList.remove("expanded");
        backdrop.classList.remove("expanded");
    } else {
        dismissModal();
        menu.classList.add("expanded");
        backdrop.classList.add("expanded");
    }
}

function dismissMobileMenu() {
    const menu = q("#mobile_menu");
    const backdrop = q("#mobile_backdrop");
    menu.classList.remove("expanded");
    backdrop.classList.remove("expanded");
}
