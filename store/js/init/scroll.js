
var scrollArrow;
var scrollAlignTop    = 0;
var scrollAlignMiddle = 1;
var scrollAlignBottom = 2;

var draggerHeightMin;
var dragging;
var dragInitMouseY;
var dragInitTop;

function initScrolls(elem) {
    
    draggerHeightMin = rem * 2;
    
    const scrolls = qAll(".scroll", elem);
    
    for (let i = 0; i < scrolls.length; i++) {
        
        let scroll = scrolls[i];
        
        const initialised = scroll.classList.contains("initialised");
        if (initialised) {
            updateScroll(scroll, true, true);
            continue;
        }
        
        const button = findAncestor(".button", scroll);
        if (button && button.parentNode.classList.contains("adder")) {
            continue;
        }
        
        let inner = document.createElement("div");
        let scrollbar = document.createElement("div");
        let dragger = document.createElement("div");
        let track = document.createElement("div");
        let btnTop = document.createElement("div");
        let btnBottom = document.createElement("div");
        
        btnTop.innerHTML = scrollArrow;
        btnBottom.innerHTML = scrollArrow;
        
        inner.classList.add("scroll_inner");
        scrollbar.classList.add("scrollbar");
        dragger.classList.add("dragger");
        track.classList.add("track");
        
        btnTop.classList.add("arrow");
        btnBottom.classList.add("arrow");
        btnTop.classList.add("top");
        btnBottom.classList.add("bottom");
        
        track.appendChild(dragger);
        scrollbar.appendChild(btnTop);
        scrollbar.appendChild(track);
        scrollbar.appendChild(btnBottom);
        
        // Element.children is a live node list. We iterate over it
        // backwards because the index of a given node will change as
        // children are removed unless said removal is done from the end.
        let children = [];
        for (let i = scroll.children.length-1; i >= 0; i--) {
            children.push(scroll.removeChild(scroll.children[i]));
        }
        for (let i = children.length-1; i >= 0; i--) {
            inner.appendChild(children[i]);
        }
        scroll.appendChild(inner);
        scroll.appendChild(scrollbar);
        scroll.classList.add("initialised");
        inner.addEventListener("scroll", scrollEvent);
        
        updateScroll(scroll, true, true);
    }
    
    window.addEventListener("mousedown", scrollMouseDown);
    window.addEventListener("mousemove", scrollHover);
}

function scrollMouseDown(e) {

    // Return if not left mouse button.
    if (e.button !== 0) {
        return;
    }
    
    let scroll = findAncestor(".scroll", e.target);
    if (!scroll) {
        return;
    }
    if (scroll.classList.contains("no_scroll")) {
        return;
    }
    
    let scrollbar = scroll.children[1];
    
    if (!inRect(scrollbar, e.clientX, e.clientY)) {
        return;
    }

    e.preventDefault();
    
    let inner = scroll.children[0];
    let dragger = q(".dragger", scrollbar);
    let track = q(".track", scrollbar);
    let arrowTop = q(".arrow.top", scrollbar);
    let arrowBottom = q(".arrow.bottom", scrollbar);
    
    if (inRect(dragger, e.clientX, e.clientY)) {
        dragStart(scroll, dragger, e.clientY);
        return;
    }
    if (inRect(track, e.clientX, e.clientY)) {
        jumpScroll(scroll, e.clientY);
        return;
    }
    if (inRect(arrowTop, e.clientX, e.clientY)) {
        animate(inner, "scrollTop", 0.25, inner.scrollTop - scrollGap, timingEase);
        return;
    }
    if (inRect(arrowBottom, e.clientX, e.clientY)) {
        animate(inner, "scrollTop", 0.25, inner.scrollTop + scrollGap, timingEase);
        return;
    }
}

function dragStart(scroll, dragger, y) {
    dragging = scroll;
    dragInitMouseY = y;
    dragInitTop = parseFloat(dragger.style.top.slice(0, -2));
    dragger.parentNode.classList.add("active");
    window.addEventListener("mousemove", dragMove);
    window.addEventListener("mouseup", dragEnd);
}
    
function dragMove(e) {
    
    let scroll = dragging;
    let inner = scroll.children[0];
    let scrollbar = scroll.children[1];
    let dragger = q(".dragger", scrollbar);
    let track = q(".track", scrollbar);

    let delta = e.clientY - dragInitMouseY;
    let offset = dragInitTop + delta;
    let dragRange = track.clientHeight - dragger.clientHeight;
    if (offset < 0) {
        offset = 0;
    }
    if (offset > dragRange) {
        offset = dragRange;
    }
    
    let pos = offset / dragRange;
    let scrollTop = (inner.scrollHeight - inner.clientHeight) * pos;
    requestAnimationFrame(function dragMoveAnimationFrame() {
        inner.scrollTop = scrollTop;
    });
}

function dragEnd(e) {
    q(".track", dragging.children[1]).classList.remove("active");
    window.removeEventListener("mousemove", dragMove);
    window.removeEventListener("mouseup", dragEnd);
    dragging = null;
}

function scrollTo(elem, options) {

    elem.classList.add("scrolled_to");
    setTimeout(() => {
        elem.classList.remove("scrolled_to");
    }, 2000 + 50);

    let alignment, disableAnim, padding;
    if (options) {
        alignment = options.alignment;
        disableAnim = options.disableAnim;
        padding = options.padding;
    }
    
    const scroll = findAncestor(".scroll", elem);
    if (!scroll) {
        return;
    }
    
    const inner = scroll.children[0];
    
    const scrollOffset = scroll.getBoundingClientRect().top;
    const scrollHeight = scroll.clientHeight;
    const innerOffset = inner.getBoundingClientRect().top;
    
    /*
        At a minimum we compensate for
        the scroll top/bottom gradients.
    */
    if (!padding) {
        padding = rem * 1.5; // compensate for the scroll top/bottom gradients
    }
    
    let y;
    if (!alignment) {
        alignment = scrollAlignTop;
    }
    switch (alignment) {
    case scrollAlignTop:
        y = elem.getBoundingClientRect().top;
        y = (inner.scrollTop + y) - scrollOffset;
        y -= padding;
        break;
    case scrollAlignBottom:
        y = elem.getBoundingClientRect().bottom;
        y = (inner.scrollTop + y) - scrollOffset;
        y -= scrollHeight;
        y += padding;
        break;
    default:
        throw "scrollTo: invalid alignment: " + alignment;
    }
    
    if (disableAnim) {
        inner.scrollTop = y;
    } else {
        animate(inner, "scrollTop", 0.4, y, timingEase);
    }
}

function jumpScroll(scroll, y) {
    
    let inner = scroll.children[0];
    let scrollbar = scroll.children[1];
    let dragger = q(".dragger", scrollbar);
    let track = q(".track", scrollbar);
    
    let r = track.getBoundingClientRect();
    let offset = (y - r.top) - (dragger.clientHeight / 2);
    
    let dragRange = r.height - dragger.clientHeight;
    if (offset < 0) {
        offset = 0;
    }
    if (offset > dragRange) {
        offset = dragRange;
    }
    
    let pos = offset / dragRange;
    let scrollableRange = inner.scrollHeight - inner.clientHeight;
    
    animate(inner, "scrollTop", 0.25, scrollableRange * pos, timingEase);
}

function scrollHover(e) {
    
    let scrollbars = qAll(".scrollbar");
    
    for (let i = 0; i < scrollbars.length; i++) {
     
        let scrollbar = scrollbars[i];
        
        q(".track", scrollbar).classList.remove("hover");
        q(".arrow.top", scrollbar).classList.remove("hover");
        q(".arrow.bottom", scrollbar).classList.remove("hover");
    }
    
    let scroll = findAncestor(".scroll", e.target);
    if (!scroll) {
        document.body.style.cursor = "";
        return;
    }
    
    if (scroll.classList.contains("no_scroll")) {
        document.body.style.cursor = "";
        return;
    }
    
    let scrollbar = scroll.children[1];
    let hovering = false
    if (hoverElem(e, q(".track", scrollbar))) hovering = true;
    if (hoverElem(e, q(".arrow.top", scrollbar))) hovering = true;
    if (hoverElem(e, q(".arrow.bottom", scrollbar))) hovering = true;
    
    if (hovering) {
        document.body.style.cursor = "pointer";
    } else {
        document.body.style.cursor = "";
    }
}

function hoverElem(e, elem) {
    if (inRect(elem, e.clientX, e.clientY)) {
        elem.classList.add("hover");
        return true;
    }
    elem.classList.remove("hover");
    return false
}

function scrollEvent(e) {
    updateScroll(e.target.parentNode, false);
    if (!scriptLoaded) {
        return;
    }
    updateEditorToolsPos(e.target);
    updateCanvasPos();
    restoreSelection();
    updateSelection(e.target);
}

function updateScrolls(e) {
    let scrolls = qAll(".scroll.initialised");
    for (let i = 0; i < scrolls.length; i++) {
        updateScroll(scrolls[i], true, true);
    }
}

function updateScroll(scroll, anim, heightChanged) {
    
    if (!scroll) {
        throw "No scroll supplied."
    }
    
    let inner = scroll.children[0];
    let scrollbar = scroll.children[1];
    let track = q(".track", scrollbar);
    let dragger = q(".dragger", scrollbar);
        
    if (inner.scrollHeight <= inner.clientHeight) {
        if (heightChanged) {
            scroll.classList.add("no_scroll");
        }
        dragger.style.height = draggerHeightMin + "px";
        return;
    }
    
    scroll.classList.remove("no_scroll");
    
    /*
        Wait until the next frame when the changes from
        removing "no_scroll" have taken effect.
    */
    requestAnimationFrame(function() {
        let draggerHeight = track.clientHeight * (inner.clientHeight / inner.scrollHeight);

        if (draggerHeight < draggerHeightMin) {
            draggerHeight = draggerHeightMin;
        }
        
        let scrollPosition = inner.scrollTop / (inner.scrollHeight - inner.clientHeight);
        let top = (track.clientHeight - draggerHeight) * scrollPosition;
        if (anim) {
            dragger.style.transitionDuration = "";
        } else {
            dragger.style.transitionDuration = "0s";
        }

        dragger.style.height = draggerHeight + "px";
        dragger.style.top = top + "px";
    });
}
