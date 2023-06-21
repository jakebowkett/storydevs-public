
function forumCategory(e) {

    const item = findAncestor(".btn", e.target);
    if (!item) {
        return;
    }
    
    e.preventDefault();
    e.stopImmediatePropagation();
    
    const browse = q("#browse .col_inner");
    browse.innerHTML = loading;
    
    let route = item.getAttribute("href");
    
    const parts = route.split("?");
    const path = parts[0];
    const query = "?"+parts[1];

    const c = context;
    c.query = query;
    c.layout = "browse";

    history.pushState({
        kind:    "mode",
        view:    c.view,
        subView: c.subView,
        query:   c.query,
        layout:  c.layout,
    }, "", route);
    setLayout("browse", c.view);

    const form = findAncestor("form", item);
    const items = qAll(".btn", form);
    for (const item of items) {
        item.classList.remove("selected");
    }
    item.classList.add("selected");
    
    get(`${path}/partial${query}`, function(err, res) {
        if (err) {
            log(err);
            return;
        }
        browse.innerHTML = res.col;
        init(q("#browse"));
    });
}

function focusReply(e) {
    e.preventDefault();
    const p = findAncestor(".post", e.currentTarget);
    history.pushState({}, "", `#${p.id}`);
    toClipboard(document.location.href);
    scrollToPost();
}

function scrollToPost() {
    const hash = (document.location.hash);
    if (hash === "") {
        return;
    }
    const post = q(hash);
    scrollTo(post, {padding: scrollToPad});
}