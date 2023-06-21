
function portfolioImageFullSize(e) {
    const gfx = findAncestor(".graphic", e.currentTarget);
    const url = q("img", gfx).getAttribute("src");
    const full = q("#full");
    const img = q("img", full);
    img.setAttribute("src", url);
    full.classList.remove("hidden");
    window.addEventListener("keydown", escapePortfolio);
}

function escapePortfolio(e) {
    if (e.key === "Escape") {
        dismissFullSize(e);
    }
    window.removeEventListener("keydown", escapePortfolio);
}

function dismissFullSize(e) {
    if (findAncestor("img", e.target)) {
        return;
    }
    const full = q("#full");
    const img = q("img", full);
    full.classList.add("hidden");
    img.removeAttribute("src");
}

function portfolioPrev(e) {
    switchPortfolioExample(e.currentTarget, "prev");
}
function portfolioNext(e) {
    switchPortfolioExample(e.currentTarget, "next");
}

function switchPortfolioExample(elem, dir) {
    
    const gfx = findAncestor(".graphic", elem);
    const ex = findAncestor(".example", gfx);
    const img = q("img", gfx);
    const urls = ex.dataset.examples.slice(0, -1).split(",");
    const aspects = ex.dataset.aspects.slice(0, -1).split(",");
    
    for (let i = 0; i < aspects.length; i++) {
        aspects[i] = parseFloat(aspects[i]);
    }
    
    let inView;
    for (let i = 0; i < urls.length; i++) {
        const current = urls[i];
        const src = img.getAttribute("src");
        if (current === src) {
            inView = i;
            break;
        }
    }
    
    if (dir === "prev") {
        inView--;
    } else {
        inView++;
    }
    
    const first = inView === 0;
    const last  = inView === urls.length-1;
    const prev = q(".prev", gfx);
    const next = q(".next", gfx);
    
    if (first) {
        prev.classList.add("hidden");
    } else {
        prev.classList.remove("hidden");
    }
    if (last) {
        next.classList.add("hidden");
    } else {
        next.classList.remove("hidden");
    }
    
    const sect = findAncestor(".section", ex);
    const meta = qAll(".meta > .inner", sect);
    for (let i = 0; i < meta.length; i++) {
        const m = meta[i];
        if (i === inView) {
            m.classList.remove("hidden");
            continue;
        }
        m.classList.add("hidden");
    }
    
    const loading = q(".loading", gfx);
    const full = q(".full", gfx);
    
    img.onload = () => {
        
        loading.classList.add("hidden");
        full.classList.remove("hidden");
        
        // 1.77 == 16:9 aspect
        if (aspects[inView] < 1.77) {
            img.classList.add("portrait");
        } else {
            img.classList.remove("portrait");
        }
    }
    
    img.src = "";
    full.classList.add("hidden");
    loading.classList.remove("hidden");
    img.src = urls[inView];
}

function switchPortfolioTab(e) {

    const tabs = e.currentTarget;
    const tab = e.target;
    
    // Return if the user has clicked on the container.
    if (tab === tabs) {
        return;
    }
    
    if (tab.classList.contains("selected")) {
        return;
    }
    
    let selected;
    for (let i = 0; i < tabs.children.length; i++) {
        const current = tabs.children[i];
        if (current === tab) {
            selected = i;
            current.classList.add("selected");
            continue;
        }
        current.classList.remove("selected");
    }
    
    const ad = findAncestor(".advertised", tabs);
    const sections = qAll(".portfolio > .section");
    
    let toHide;
    let toShow;
    
    for (let i = 0; i < sections.length; i++) {
        const current = sections[i];
        if (i === selected) {
            toShow = current;
            continue;
        }
        if (!current.classList.contains("hidden")) {
            toHide = current;
        }
    }
    
    toShow.classList.remove("hidden");
    toHide.classList.add("hidden");
}