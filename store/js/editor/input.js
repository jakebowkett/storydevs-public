
// We use this to handle keyboard-specific input
// such as arrow keys, home/end, ctrl+[letter],
// and so on that aren't covered by the input
// event.
function editorKeydown(e) {
    
    let keepNextFormat = false;
    
    switch (e.key) {
    
    case "Tab":
        editorUnfocus();
        return;
        
    case "ArrowUp":
    case "ArrowDown":
        e.preventDefault();
        if (e.shiftKey) {
            navVertical(e.key === "ArrowUp", true);
            break;
        }
        navVertical(e.key === "ArrowUp");
        break;
        
    case "ArrowLeft":
    case "ArrowRight":
        e.preventDefault();
        if (e.ctrlKey && e.shiftKey) {
            toWordBoundary(e.key === "ArrowLeft", true);
            break;
        }
        if (e.ctrlKey) {
            toWordBoundary(e.key === "ArrowLeft");
            break;
        }
        if (e.shiftKey) {
            navHorizontal(e.key === "ArrowLeft", true);
            break;
        }
        navHorizontal(e.key === "ArrowLeft");
        break;
        
    case "Home":
    case "End":
        e.preventDefault();
        if (e.ctrlKey && e.shiftKey) {
            toEditorBoundary(e.key === "Home", true);
            break;
        }
        if (e.ctrlKey) {
            toEditorBoundary(e.key === "Home");
            break;
        }
        if (e.shiftKey) {
            toLineBoundary(e.key === "Home", true);
            break;
        }
        toLineBoundary(e.key === "Home");
        break;
    default:
    
        if (!e.ctrlKey) {
            return;
        }

        switch (e.key) {
            
        /*
            We have this empty case statement here because
            otherwise users might expect that ctrl+u will
            result in underline formatting but on Chrome
            it just opens the page source code.
        */
        case "u":
            e.preventDefault();
            break;
            
        case "b":
        case "i":
            e.preventDefault();
            toggleInlineFormat(e.key);
            keepNextFormat = true;
            break;
        
        /*    
            We return for links because it brings up a new
            dialogue. The stuff below doesn't need to occur
            as it does with normal nav and formatting. In
            particular, the code below updates the caret which
            we actually want to remove while editing links.
        */
        case "k":
            e.preventDefault();
            addLink();
            return;

        case "h":
            e.preventDefault();
            toggleAtomic("h2");
            break;
        case "q":
            e.preventDefault();
            toggleAtomic("blockquote");
            break;

        case "a":
            e.preventDefault();
            selectAll();
            break;
        }
        
        // ctrl + shift + 7/8
        if (e.shiftKey && (e.keyCode === 55 || e.keyCode === 56)) {
            e.preventDefault();
            toggleList(e.keyCode === 55);
        }
    }
    
    // Clear caret x memory unless we're going up or down.
    if (e.key !== "ArrowUp" && e.key !== "ArrowDown") {
        clearCaretXMem();
    }
    
    afterEditorInteraction();
    if (!keepNextFormat) {
        clearNextFormat();
    }
}

function afterEditorInteraction() {
    const selection = orderedSelection();
    const start = selection.start;
    updateFormatMenu(start.node, start.idx);
    updateSelection();
    dismissLinkEditor();
    updatePlaceholderText();
    updateScroll(findAncestor(".scroll", focusedEditor), true, true);
    ensureCaretInView();
}

function ensureCaretInView(padding) {
    
    const caret = q("#caret");
    const scroll = findAncestor(".scroll", focusedEditor);
    const cRect = caret.getBoundingClientRect();
    const sRect = scroll.getBoundingClientRect();
    
    /*
        At a minimum we compensate for
        the scroll top/bottom gradients.
    */
    if (!padding) {
        padding = rem * 1.5; 
    }
    
    let needToScroll;
    let alignment = scrollAlignTop;
    if ((cRect.top - padding) < sRect.top) {
        needToScroll = true;
    }
    if ((cRect.bottom + padding) > sRect.bottom) {
        needToScroll = true;
        alignment = scrollAlignBottom;
    }
    
    if (needToScroll) {
        scrollTo(caret, {
            alignment: alignment,
            disableAnim: true,
            padding: padding,
        });
    }
}

function clearCaretXMem() {
    ed.focus.xMem = ed.focus.caretX;
    ed.anchor.xMem = ed.anchor.caretX;
}

function updatePlaceholderText() {
    const input = q(".input", focusedEditor);
    const tag = tagName(input.children[0]);
    const empty = input.textContent.trim() === edEmptyChar;
    const show = tag === "p" && empty;
    const placeholder = q(".bg p");
    if (show) {
        placeholder.style.display = "";
    } else {
        placeholder.style.display = "none";
    }
}

function clearNextFormat() {
    ed.addFormat = [];
    ed.removeFormat = [];
    const focus = ed.focus;
    updateFormatMenu(focus.node, focus.idx);
}

function addLink() {
    if (selectionCollapsed()) {
        return;
    }
    removeCaret();
    editorRemoveEvents();
    showLinkEditor(true, false);
}

function viewLink() {
    
    // We use anchor because it's possible
    // for focus to be an empty object if
    // the user has moused down but not moved
    // or moused up.
    const anchor = ed.anchor;
    const ri = new RuneIterator(anchor.node, anchor.idx);
    const r = ri.current();
    const linkClicked = findAncestor(".a", r.node);
    if (!linkClicked) {
        return;
    }
    
    const input = q("#link_editor input");
    input.value = linkClicked.dataset.link;
    
    showLinkEditor(false, true);
}

function linkKeydown(e) {
    switch (e.key) {
    case "Enter":
        e.preventDefault();
        applyLink(e);
        break;
    case "Escape":
        e.preventDefault();
        dismissLinkEditor();
        break;
    }
}

function showLinkEditor(focus, edit) {
    
    const container = q("#link_editor");
    const input = q("input", container);
    
    if (edit) {
        q(".apply", container).style.display = "none";
        q(".edit", container).style.display = "";
        q(".remove", container).style.display = "";
    } else {
        q(".apply", container).style.display = "";
        q(".edit", container).style.display = "none";
        q(".remove", container).style.display = "none";
    }
    
    container.classList.add("visible");
    
    if (focus) {
        input.focus();
    }
    
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    const edRect = focusedEditor.getBoundingClientRect();
    const conRect = container.getBoundingClientRect();
    
    const winX = window.innerWidth;
    const winY = window.innerHeight;
    const yBoundary = edRect.bottom < winY ? edRect.bottom : winY;
    
    if ((start.caretX + conRect.width) > winX) {
        container.style.right = (winX - end.caretX) + "px";
        container.style.left = "";
    } else {
        container.style.left = start.caretX + "px";
        container.style.right = "";
    }
    
    const offsetY = end.caretY + end.caretHeight;
    if ((offsetY + conRect.height) > yBoundary) {
        container.style.bottom = (winY - start.caretY) + "px";
        container.style.top = "";
    } else {
        container.style.top = offsetY + "px";
        container.style.bottom = "";
    }

    window.addEventListener("keydown", linkKeydown);
}

function focusLinkEditor(e) {
    if (findAncestor(".btn", e.target)) {
        return;
    }
    e.preventDefault();
    removeCaret();
    editorRemoveEvents(); 
}

function applyLink(e) {
    e.preventDefault();
    const container = q("#link_editor");
    const input = q("input", container);
    const link = input.value.trim();
    if (link.length > 0) {
        toggleInlineFormat("a", (span) => {
            span.dataset.link = input.value.trim();
        });
    }
    restoreSelection();
}

function expandToLink() {
    
    const anchor = ed.anchor;
    const focus = ed.focus;
    const ri = new RuneIterator(anchor.node, anchor.idx);
    let r = ri.current();
    const nodes = ri.nodes;
    const n = ri.n;
    const link = r.node.parentNode.dataset.link;
    
    let linkStart = anchor.idx - r.offset;
    let linkEnd = linkStart + r.node.textContent.length;
    
    for (let i = n-1; i >= 0; i--) {
        const span = nodes[i].parentNode;
        if (span.dataset.link === link) {
            linkStart -= span.textContent.length;
        }
    }
    
    for (let i = n+1; i < nodes.length; i++) {
        const span = nodes[i].parentNode;
        if (span.dataset.link === link) {
            linkEnd += span.textContent.length;
        }
    }
    
    const anchorLine = anchor.line;
    const anchorLastLine = anchor.lastLine;
    const anchorIdx = anchor.idx;
    
    const focusLine = focus.line;
    const focusLastLine = focus.lastLine;
    const focusIdx = focus.idx;
    
    const startLine = lineFromPos(ri, linkStart);
    anchor.line = startLine.num;
    anchor.lastLine = startLine.last;
    anchor.idx = linkStart;
    
    const endLine = lineFromPos(ri, linkEnd);
    focus.line = endLine.num;
    focus.lastLine = endLine.last;
    focus.idx = linkEnd;
    
    return {
        anchorLine: anchorLine,
        anchorLastLine: anchorLastLine,
        anchorIdx: anchorIdx,
        focusLine: focusLine,
        focusLastLine: focusLastLine,
        focusIdx: focusIdx,
    };
}

function editLink(e) {
    e.preventDefault();
    const meta = expandToLink();
    const container = q("#link_editor");
    const input = q("input", container);
    const link = input.value.trim();
    if (link.length > 0) {
        toggleInlineFormat("a", (span) => {
            span.dataset.link = input.value.trim();
        });
    }
    restoreSelection(meta);
}

function removeLink(e) {
    e.preventDefault();
    const meta = expandToLink();
    toggleInlineFormat("a", null, (span) => {
        span.removeAttribute("data-link");
    });
    restoreSelection(meta);
}

function restoreSelection(meta) {
    
    // We add this check so that it can be
    // called from scrollEvent in scrolls.js
    if (!focusedEditor) {
        return;
    }
    
    if (meta) {
        const anchor = ed.anchor;
        const focus = ed.focus;

        anchor.line = meta.anchorLine;
        anchor.lastLine = meta.anchorLastLine;
        anchor.idx = meta.anchorIdx;
        
        focus.line = meta.focusLine;
        focus.lastLine = meta.focusLastLine;
        focus.idx = meta.focusIdx;
    }
    
    dismissLinkEditor();
    editorAddEvents();
    updateAfterFormatButton();
    syncTextarea();
}

function lineFromPos(ri, idx) {
    
    ri.seek(idx);
    const r = ri.current();
    
    const range = document.createRange();
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    let rect = range.getBoundingClientRect();
    
    const line = lineFromRune(r.node.parentNode, rect);
    
    return {
        num: line.atRune,
        last: line.last,
    };
}

function dismissLinkEditor() {
    const container = q("#link_editor");
    const input = q("input", container);
    container.classList.remove("visible");
    input.value = "";

    window.removeEventListener("keydown", linkKeydown);
}

function editorInput(e) {
    
    // const o = {};
    // for (const prop in e) {
    //     const t = typeof e[prop];
    //     if (t !== "string" && t !== "number") {
    //         continue;
    //     }
    //     o[prop] = e[prop];
    // }
    
    // postJson("/debug", o, function(err, res) {
    //     if (err) {
    //         log(err);
    //     }
    // });
    
    if (!e.data) {
        switch (e.inputType) {
        case "insertLineBreak":
            newParagraph();
            break;
        case "deleteContentBackward":
        case "deleteContentForward":
            if (selectionCollapsed()) {
                editorBackspace();
            } else {
                editorDelete();
            }
            break;
        }
        clearCaretXMem();
        afterEditorInteraction();
        clearNextFormat();
        return;
    }
    
    /*
        TODO:
            The textarea caret needs to match the custom editor
            caret so that composition suggestions are appropriate.
    */
    if (e.inputType === "insertCompositionText") {
        toWordBoundary(true);
        toWordBoundary(false, true);
    }
    
    editorDelete();
    editorInsert(e.data);
    
    // This code prevents a link being extended when you
    // type at the end of it.
    const p = ed.focus.node;
    const tc = p.textContent;
    const ri = new RuneIterator(p, ed.focus.idx);
    const r = ri.current();
    const atTextNodeBoundary = r.offset === 0 || r.overall === tc.length;
    if (atTextNodeBoundary && r.overall !== 0) {
        ri.prev();
        const span = ri.current().node.parentNode;
        if (span.classList.contains("a")) {
            navHorizontal(true, true);
            toggleInlineFormat("a", null, (span) => {
                span.removeAttribute("data-link");
            });
            navHorizontal(false);
        }
    }
    
    // If there's no formatting changes to
    // be made we return.
    let af = ed.addFormat;
    let rf = ed.removeFormat;
    if (af.length === 0 && rf.length === 0) {
        clearCaretXMem();
        afterEditorInteraction();
        clearNextFormat();
        return;
    }
    
    // Otherwise we select the rune we just
    // inserted then add and remove formats
    // as necessary. We use empty functions
    // to force off the usual toggling.
    navHorizontal(true, true);
    for (const f of af) {
        toggleInlineFormat(f, () => {});
    }
    for (const f of rf) {
        toggleInlineFormat(f, null, () => {});
    }
    
    // We then deselect the rune and clear
    // the add/remove format lists.
    navHorizontal(false);
    clearNextFormat();
    
    clearCaretXMem();
    afterEditorInteraction();
    clearNextFormat();
}

function newParagraph() {
    
    // After the call to editorDelete() the
    // selection is guaranteed to be collapsed.
    editorDelete();
    
    const anchor = ed.anchor;
    const focus = ed.focus;
    
    // Empty list item.
    if (focus.pEmpty && tagName(focus.node) === "li") {
        
        const newListItems = [];
        const list = focus.node.parentNode;
        let seen = false;
        for (const item of Array.from(list.children)) {
            if (item === focus.node) {
                seen = true;
                continue;
            }
            if (seen) {
                newListItems.push(item);
            }
        }
        list.removeChild(focus.node);
        
        const p = document.createElement("p");
        const span = document.createElement("span");
        span.textContent = edEmptyChar;
        p.appendChild(span);
        insertAfter(p, list);
        
        focus.node = p;
        focus.paragraphs[focus.p] = p;
        anchor.node = p;
        anchor.paragraphs[focus.p] = p;
        
        const newList = list.cloneNode(false);
        for (const item of newListItems) {
            newList.appendChild(item);
        }
        insertAfter(newList, p);
        removeEmptyLists();
        
        return;
    }
    
    // At the start of a paragraph.
    if (focus.idx === 0) {
        
        let tag = tagName(focus.node);
        if (contains(editorAtomics, tag) && focus.pEmpty) {
            tag = "p";
        }
        
        const newP = document.createElement(tag);
        const span = document.createElement("span");
        span.textContent = edEmptyChar;
        newP.appendChild(span);
        focus.node.parentNode.insertBefore(newP, focus.node);
        focus.paragraphs.splice(focus.p, 0, newP);
        
        focus.p++;
        focus.caretY = focus.node.getBoundingClientRect().top;
        
        Object.assign(anchor, focus);
        placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
        
        return;
    }
    
    // At the end of a paragraph.
    if (focus.idx === focus.node.textContent.length) {
        
        let tag = tagName(focus.node);
        if (contains(editorAtomics, tag)) {
            tag = "p";
        }
        
        const newP = document.createElement(tag);
        const span = document.createElement("span");
        span.textContent = edEmptyChar;
        newP.appendChild(span);
        
        if (focus.lastP) {
            focus.node.parentNode.append(newP);
        } else {
            insertAfter(newP, focus.node);
        }
        focus.paragraphs.splice(focus.p+1, 0, newP);
        
        const rect = newP.getBoundingClientRect();
        
        focus.node = newP;
        focus.pEmpty = true;
        focus.p++;
        focus.line = 0;
        focus.lastLine = true;
        focus.idx = 0;
        focus.caretX = rect.left;
        focus.caretY = rect.top;
        
        Object.assign(anchor, focus);
        placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
        
        return;
    }
    
    // In the middle of a paragraph.
    // const tag = focus.node.tagName.toLowerCase();
    const newP = focus.node.cloneNode(true);
    
    // Truncate original paragraph to its first half.
    let ri = new RuneIterator(focus.node, focus.idx);
    let r = ri.current();
    let tnToRemove = ri.nodesAfter(r.node);
    for (const tn of tnToRemove) {
        removeTextNode(tn);
    }
    r.node.textContent = r.node.textContent.slice(0, r.offset);
    if (r.node.textContent.length === 0) {
        removeTextNode(r.node);
    }
    
    // Truncate new paragraph to its last half.
    ri = new RuneIterator(newP, focus.idx);
    r = ri.current();
    tnToRemove = ri.nodesBefore(r.node);
    for (const tn of tnToRemove) {
        removeTextNode(tn);
    }
    const tc = r.node.textContent;
    r.node.textContent = tc.slice(r.offset, tc.length);
    
    if (focus.lastP) {
        focus.node.parentNode.append(newP);
    } else {
        insertAfter(newP, focus.node);
    }
    
    const rect = newP.getBoundingClientRect();
    
    focus.paragraphs.splice(focus.p+1, 0, newP);
    focus.node = newP;
    focus.p++;
    focus.line = 0;
    focus.lastLine = rect.height === focus.caretHeight;
    focus.idx = 0;
    focus.caretX = rect.left;
    focus.caretY = rect.top;
    
    Object.assign(anchor, focus);
    placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
}

function textNodeRoot(node) {
    
    while (true) {
        
        const tag = node.parentNode.tagName.toLowerCase();
        if (contains(editorParagraphs, tag)) {
            return node;
        }
        
        node = node.parentNode;
    }
}

function editorInsert(s) {
    
    s = s.replace(/[\u2400-\u243F]/, "");
    
    const range = document.createRange();
    const focus = ed.focus;
    const anchor = ed.anchor;
    const ri = new RuneIterator(focus.node, focus.idx);
    const r = ri.current();
    
    if (r.overall !== 0 && r.offset === 0) {
        ri.prev();
        const prevRune = ri.current();
        prevRune.node.textContent += s;
    } else {
        const tc = r.node.textContent;
        let before = tc.slice(0, r.offset);
        let after = tc.slice(r.offset, tc.length);
        if (focus.node.textContent === edEmptyChar) {
            after = "";
        }
        r.node.textContent = before + s + after;
    }
    
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + s.length);
    const rects = range.getClientRects();
    const rect = rects[rects.length-1];
    const line = lineFromRune(focus.node, rect);
    
    focus.line = line.atRune;
    focus.idx += s.length;
    focus.caretX = rect.right;
    focus.caretY = line.top;
    focus.pEmpty = false;
    
    Object.assign(anchor, focus);
    placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
}


function editorCut(e) {
    
    e.preventDefault();
    
    const ta = e.target;
    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    ta.value = ta.value.slice(0, start) + ta.value.slice(end);
    
    if (selectionCollapsed()) {
        return;
    }
    
    e.clipboardData.setData("text/plain", selectedText());
    
    editorDelete();
}

function editorCopy(e) {

    // If copy is occurring in link editor allow it as usual.
    if (findAncestor("#link_editor", e.target)) {
        return;
    }
    
    e.preventDefault();
    
    if (selectionCollapsed()) {
        return;
    }

    e.clipboardData.setData("text/plain", selectedText());
}

function selectedText() {
    
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    
    // Same paragraph.
    if (start.p === end.p) {
        
        const p = start.node;
        
        if (start.idx === 0 && end.p === 0) {
            return p.textContent;
        }
        
        const ri = new RuneIterator(p, start.idx);
        let r = ri.current();
        const tnStart = r.node;
        const offsetStart = r.offset;
        
        ri.seek(end.idx);
        r = ri.current();
        const tnEnd = r.node;
        const offsetEnd = r.offset;
        
        const between = ri.nodesBetween(tnStart, tnEnd);
        
        if (tnStart === tnEnd) {
            return tnStart.textContent.slice(offsetStart, offsetEnd);
        }
        
        let tc = "";
        tc += tnStart.textContent.slice(offsetStart);
        for (const tn of between) {
            tc += tn.textContent;
        }
        tc += tnEnd.textContent.slice(0, offsetEnd);
        
        return tc;
    }
    
    // Different paragraphs.
    let tc = "";
    let ri = new RuneIterator(start.node, start.idx);
    let r = ri.current();
    tc += r.node.textContent.slice(r.offset, r.node.textContent.length);
    let after = ri.nodesAfter(r.node);
    for (const tn of after) {
        tc += tn.textContent;
    }

    const middleParagraphs = end.paragraphs.slice(start.p+1, end.p);
    for (const p of middleParagraphs) {
        tc += "\n" + p.textContent;
    }
    tc += "\n";
    
    ri = new RuneIterator(end.node, end.idx);
    r = ri.current();
    let before = ri.nodesBefore(r.node);
    for (const tn of before) {
        tc += tn.textContent;
    }
    tc += r.node.textContent.slice(0, r.offset);
    
    return tc;
}

function editorPaste(e) {
    
    
    e.preventDefault();
    
    const s = e.clipboardData.getData("text");
    const parts = [];
    
    for (let p of s.split("\n")) {
        p = p.trim();
        if (p === "") {
            continue;
        }
        parts.push(p);
    }
    
    if (parts.length === 0) {
        return;
    }
    
    const ta = e.target;
    const taStart = ta.selectionStart;
    const taEnd = ta.selectionEnd;
    ta.value = ta.value.slice(0, taStart) + parts.join("\n") + ta.value.slice(taEnd);
    
    if (parts.length === 1) {
        editorDelete();
        editorInsert(parts[0]);
        updatePlaceholderText();
        return;
    }
    
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    const tag = tagName(start.node);
    
    editorInsert(parts.shift());
    newParagraph();
    editorInsert(parts.pop());
    
    for (const part of parts) {
        
        const p = document.createElement(tag);
        const span = document.createElement("span");
        span.textContent = part;
        p.appendChild(span);
        
        insertBefore(p, end.node);
        
        start.paragraphs.splice(end.p, 0, p);
        end.paragraphs = start.paragraphs;
        
        start.p++;
        end.p++;
    }
    
    updateSelection();
    updatePlaceholderText();
}

function mergeParagraphs(p1, p2) {
    
    // When merging to atomic paragraphs
    // we remove any inline styling.
    const tag = p1.tagName.toLowerCase();
    if (contains(editorAtomics, tag)) {
        
        // We modify p1.childNode[0].textContent rather
        // than p1.textContent because the latter will
        // erase p1's reference to it's current child
        // nodes. Instances of RuneIterator may still
        // be holding references to such nodes.
        p1.childNodes[0].textContent += p2.textContent;
        return;
    } 
    
    const tnLast = lastTextNode(p1);
    const tnFirst = firstTextNode(p2);
    if (sameInlineFormatting(tnLast, tnFirst)) {
        tnLast.textContent += tnFirst.textContent;
        removeTextNode(tnFirst);
    }
    p1.append(...p2.childNodes);
}

// editorBackspace operates on the assumption that
// it is working with a collapsed selection - i.e.
// that anchor and focus are the same.
function editorBackspace() {
    
    const range = document.createRange();
    const focus = ed.focus;
    const anchor = ed.anchor;
    
    // First paragraph, first index.
    if (focus.p === 0 && focus.idx === 0) {
        placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
        return;
    }
    
    // Paragraph is empty.
    if (focus.pEmpty) {
        
        // Remove current paragraph.
        const p = focus.paragraphs[focus.p];
        p.parentNode.removeChild(p);
        
        // Place cursor at end of previous paragraph.
        const newP = focus.paragraphs[focus.p-1];
        
        let idx;
        if (newP.textContent === edEmptyChar) {
            idx = 0;
        } else {
            idx = newP.textContent.length;
        }
        
        const ri = new RuneIterator(newP, idx);
        const r = ri.current();
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset);
        const rect = range.getBoundingClientRect();
        const line = lineFromRune(newP, rect);
        
        focus.paragraphs.splice(focus.p, 1);
        focus.node = newP;
        focus.pEmpty = newP.textContent === edEmptyChar;
        focus.p--;
        focus.lastP = focus.p === focus.paragraphs.length-1;
        focus.line = line.atRune;
        focus.lastLine = line.last;
        focus.idx = idx;
        focus.caretX = rect.right;
        focus.caretY = line.top;
        focus.caretHeight = line.height;
        
        Object.assign(anchor, focus);
        placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
        
        removeEmptyLists();
        combineLists();
        
        return;
    }
    
    // First index.
    if (focus.idx === 0) {
        
        const prevP = focus.paragraphs[focus.p-1];
        
        // Preceding paragraph is empty.
        if (prevP.textContent === edEmptyChar) {
            
            focus.paragraphs.splice(focus.p-1, 1);
            focus.p--;
            
            prevP.parentNode.removeChild(prevP);
            
            const currentP = focus.paragraphs[focus.p];
            const rect = currentP.getBoundingClientRect();
            
            focus.caretX = rect.left;
            focus.caretY = rect.top;
            
            Object.assign(anchor, focus);
            placeCaretAt(rect.left, rect.top, focus.caretHeight);
            
            return;
        }
        
        // Join the first text node of the current
        // paragraph with the last text node of the
        // previous paragraph if they have the same
        // in-line formatting (e.g., ignore list,
        // blockquote, etc styling when comparing
        // them.
        const prevLen = prevP.textContent.length;
        const currentP = focus.node;
        
        mergeParagraphs(prevP, currentP);
        
        const ri = new RuneIterator(prevP, prevLen);
        const r = ri.current();
        
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset + r.len);
        
        const rect = range.getBoundingClientRect();
        const line = lineFromRune(prevP, rect);
        
        focus.paragraphs.splice(focus.p, 1);
        focus.node = prevP;
        focus.pEmpty = prevP.textContent === edEmptyChar;
        focus.p--;
        focus.lastP = focus.p === focus.paragraphs.length-1;
        focus.line = line.atRune;
        focus.lastLine = line.last;
        focus.idx = prevLen;
        focus.caretX = rect.left;
        focus.caretY = line.top;
        focus.caretHeight = line.height;
        
        currentP.parentNode.removeChild(currentP);
        
        Object.assign(anchor, focus);
        placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
        
        removeEmptyLists();
        combineLists();
        
        return;
    }
    
    // From now on we're within the same paragraph.
    const ri = new RuneIterator(focus.node, focus.idx);
    ri.prev();
    let r = ri.current();
    const deletedLen = r.len;
    const p = focus.node;
    
    // If we're deleting the last rune in a paragraph.
    if (r.overall === 0 && r.len === p.textContent.length) {
        
        const rect = p.getBoundingClientRect();
        
        focus.idx = 0;
        focus.caretX = rect.left;
        focus.caretY = rect.top;
        focus.pEmpty = true;
        
        const span = document.createElement("span");
        span.textContent = edEmptyChar;
        p.innerHTML = "";
        p.appendChild(span);
        
        Object.assign(anchor, focus);
        placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
        
        return;
    }
    
    const tc = r.node.textContent;
    if (tc.length === deletedLen) {
        removeTextNode(r.node);
    } else {
        const before = tc.slice(0, r.offset);
        const after = tc.slice(r.offset+deletedLen, tc.length);
        r.node.textContent = before + after;
    }
    
    // If backspacing the previous character put
    // us at the start of the paragraph.
    if ((focus.idx - deletedLen) === 0) {
        
        const rect = p.getBoundingClientRect();
        
        focus.idx = 0;
        focus.caretX = rect.left;
        focus.caretY = rect.top;
        
        Object.assign(anchor, focus);
        placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
        
        return;
    }
    
    ri.prev();
    r = ri.current();
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    const rect = range.getBoundingClientRect();
    const line = lineFromRune(p, rect);
    
    focus.line = line.atRune;
    focus.idx -= deletedLen;
    focus.caretX = rect.right;
    focus.caretY = line.top;
    
    Object.assign(anchor, focus);
    placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
}

function collapseToParagraphBeginning(start, end) {
    start.lastLine = start.node.clientHeight === start.caretHeight;
    Object.assign(end, start);
    clearEditorCanvas();
    placeCaretAt(start.caretX, start.caretY, start.caretHeight);
}

function collapseToPreviousRune(start, end, ri) {
    
    ri.prev();
    const r = ri.current();
    
    const range = document.createRange();
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    const rect = range.getBoundingClientRect();
    const line = lineFromRune(start.node, rect);
    
    start.line = line.atRune;
    start.lastLine = line.last;
    start.caretX = rect.right;
    start.caretY = line.top;
    
    Object.assign(end, start);
    clearEditorCanvas();
    placeCaretAt(start.caretX, start.caretY, start.caretHeight);
}

function editorDelete() {
    
    if (selectionCollapsed()) {
        return;
    }
    
    const focus = ed.focus;
    const anchor = ed.anchor;
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    const diff = end.p - start.p;
    
    // Start and end are in the same paragraph.
    if (diff === 0) {
        
        // If deletion would make the paragraph empty.
        if (start.idx === 0 && end.idx === end.node.textContent.length) {
            
            const content = start.node.textContent;
            
            const span = document.createElement("span");
            span.textContent = edEmptyChar;
            start.node.innerHTML = "";
            start.node.appendChild(span);
            
            start.line = 0;
            start.lastLine = true;
            start.idx = 0;
            start.pEmpty = true;
            
            Object.assign(end, start);
            clearEditorCanvas();
            placeCaretAt(start.caretX, start.caretY, start.caretHeight);
            
            return;
        }
        
        // We start from the end then iterate backwards
        // so that we end up next to where the caret
        // needs to placed.
        const ri = new RuneIterator(end.node, end.idx);
        let r = ri.current();
        
        const tnEnd = r.node;
        const offsetEnd = r.offset;
        ri.seek(start.idx);
        r = ri.current();
        const tnStart = r.node;
        const offsetStart = r.offset;
        
        // If start and end are in the same text node.
        if (tnStart === tnEnd) {
            
            // If we're deleting an entire text node.
            if (offsetStart === 0 && offsetEnd === tnStart.textContent.length) {
                
                if (start.idx === 0) {
                    removeTextNode(tnStart);
                    collapseToParagraphBeginning(start, end);
                    return;
                }
                
                ri.prev();
                r = ri.current();
                
                removeTextNode(tnStart);
                
                const range = document.createRange();
                range.setStart(r.node, r.offset);
                range.setEnd(r.node, r.offset + r.len);
                const rect = range.getBoundingClientRect();
                const line = lineFromRune(r.node.parentNode, rect);
                
                start.line = line.atRune;
                start.lastLine = line.last;
                start.caretX = rect.right;
                start.caretY = line.top;
                
                Object.assign(end, start);
                clearEditorCanvas();
                placeCaretAt(start.caretX, start.caretY, start.caretHeight);
                
                return;
            }
            
            const tc = tnStart.textContent;
            const before = tc.slice(0, offsetStart);
            const after = tc.slice(offsetEnd, tc.length);
            tnStart.textContent = before + after;
            
            // If we're truncating the node at the beginning of the paragraph.
            if (start.idx === 0) {
                collapseToParagraphBeginning(start, end);
                return;
            }
            
            // If we're truncating the node anywhere else.
            collapseToPreviousRune(start, end, ri);
            return;
        }
        
        /*===========================================
            start and end are in different text
            nodes of the same paragraph.
        ============================================*/
        
        // If tnStart and tnEnd both need to be removed.
        if (offsetStart === 0 && offsetEnd === tnEnd.textContent.length) {
            
            removeTextNode(tnStart);
            removeTextNode(tnEnd);
            
            const tnToRemove = ri.nodesBetween(tnStart, tnEnd);
            for (const tn of tnToRemove) {
                removeTextNode(tn);
            }
            
            if (start.idx === 0) {
                collapseToParagraphBeginning(start, end);
                return;
            }
            
            collapseToPreviousRune(start, end, ri);
            return;
        }
        
        // If only tnStart needs to be removed.
        if (offsetStart === 0) {
            
            removeTextNode(tnStart);
            
            const tnToRemove = ri.nodesBetween(tnStart, tnEnd);
            for (const tn of tnToRemove) {
                removeTextNode(tn);
            }
            
            tnEnd.textContent = tnEnd.textContent.slice(offsetEnd);
            
            if (start.idx === 0) {
                collapseToParagraphBeginning(start, end);
                return;
            }
            
            collapseToPreviousRune(start, end, ri)
            return;
        }
        
        // If only tnEnd needs to be removed.
        if (offsetEnd === tnEnd.textContent.length) {
            
            removeTextNode(tnEnd);
            
            const tnToRemove = ri.nodesBetween(tnStart, tnEnd);
            for (const tn of tnToRemove) {
                removeTextNode(tn);
            }
            
            tnStart.textContent = tnStart.textContent.slice(0, offsetStart);
            collapseToPreviousRune(start, end, ri);
            
            return;
        }
        
        // tnStart and tnEnd need to be merged.
        if (sameInlineFormatting(tnStart, tnEnd)) {
            
            const last = tnEnd.textContent.slice(offsetEnd);
            tnStart.textContent = tnStart.textContent.slice(0, offsetStart) + last;
            
            removeTextNode(tnEnd);
            
            const tnToRemove = ri.nodesBetween(tnStart, tnEnd);
            for (const tn of tnToRemove) {
                removeTextNode(tn);
            }

            collapseToPreviousRune(start, end, ri);
            return;
        }
        
        // Both tnStart and tnEnd need to be truncated.
        const tnToRemove = ri.nodesBetween(tnStart, tnEnd);
        for (const tn of tnToRemove) {
            removeTextNode(tn);
        }
        
        tnStart.textContent = tnStart.textContent.slice(0, offsetStart);
        tnEnd.textContent = tnEnd.textContent.slice(offsetEnd);
        collapseToPreviousRune(start, end, ri);
        
        return;
    }
    
    /*===============================================
        start and end are in different paragraphs
    ================================================*/
    
    if (start.idx === 0 && end.idx === end.node.textContent.length) {
        
        const span = document.createElement("span");
        span.textContent = edEmptyChar;
        start.node.innerHTML = "";
        start.node.appendChild(span);
        
        const pToRemove = start.paragraphs.splice(start.p+1, end.p - start.p);
        for (const p of pToRemove) {
            p.parentNode.removeChild(p);
        }
        
        start.lastP = start.p === start.paragraphs.length-1;
        start.line = 0;
        start.lastLine = true;
        
        removeEmptyLists();
        combineLists();
        
        collapseToParagraphBeginning(start, end);
        
        return;
    }
    
    if (start.idx === 0) {
        
        // After this end is where start was but its
        // metadata (p number etc) haven't been updated.
        const pToRemove = start.paragraphs.splice(start.p, end.p - start.p);
        for (const p of pToRemove) {
            p.parentNode.removeChild(p);
        }
        
        const ri = new RuneIterator(end.node, end.idx);
        const r = ri.current();
        
        r.node.textContent = r.node.textContent.slice(r.offset);
        const tnToRemove = ri.nodesBefore(r.node);
        for (const tn of tnToRemove) {
            removeTextNode(tn);
        }
        
        end.paragraphs = start.paragraphs;
        end.p = start.p;
        end.line = 0;
        end.lastLine = end.node.clientHeight === end.caretHeight;
        end.idx = 0;
        end.caretX = start.caretX;
        end.caretY = start.caretY;
        
        removeEmptyLists();
        combineLists();
        
        Object.assign(start, end);
        clearEditorCanvas();
        placeCaretAt(start.caretX, start.caretY, start.caretHeight);
        
        return;
    }
    
    if (end.idx === end.node.textContent.length) {
        
        const pToRemove = start.paragraphs.splice(start.p+1, end.p - start.p);
        for (const p of pToRemove) {
            p.parentNode.removeChild(p);
        }
        
        const ri = new RuneIterator(start.node, start.idx);
        const r = ri.current();
        
        const tc = r.node.textContent;
        r.node.textContent = tc.slice(0, r.offset);
        const tnToRemove = ri.nodesAfter(r.node);
        for (const tn of tnToRemove) {
            removeTextNode(tn);
        }
        
        removeEmptyLists();
        combineLists();
        
        collapseToPreviousRune(start, end, ri);
        return;
    }
    
    // Truncate end paragraph. We do the end first
    // because ri needs to refer to start.node for
    // collapseToPreviousRune to work below.
    let ri = new RuneIterator(end.node, end.idx);
    let r = ri.current();
    r.node.textContent = r.node.textContent.slice(r.offset);
    let tnToRemove = ri.nodesBefore(r.node);
    for (const tn of tnToRemove) {
        removeTextNode(tn);
    }
    
    // Truncate start paragraph.
    ri = new RuneIterator(start.node, start.idx);
    r = ri.current();
    let tc = r.node.textContent;
    r.node.textContent = tc.slice(0, r.offset);
    tnToRemove = ri.nodesAfter(r.node);
    for (const tn of tnToRemove) {
        removeTextNode(tn);
    }
    
    // Move end child nodes to start, merging if necessary.
    mergeParagraphs(start.node, end.node);
    
    // Remove middle paragraphs.
    const pToRemove = start.paragraphs.splice(start.p+1, end.p-start.p);
    for (const p of pToRemove) {
        p.parentNode.removeChild(p);
    }
    
    removeEmptyLists();
    combineLists();
    
    collapseToPreviousRune(start, end, ri);
}


function lineFromRune(node, runeRect) {
    
    const lineHeight = editorLineHeight(node);
    const rect = node.getBoundingClientRect();
    const lineCount = rect.height / lineHeight;
    const offset = (runeRect.top + (runeRect.height / 2)) - rect.top;
    const lineAtRune = Math.floor(offset / lineHeight);
    const lineTop = rect.top + (lineHeight * lineAtRune);
    const avgLineLength = Math.round((node.textContent.length-1) / lineCount);
    
    return {
        top: lineTop,
        last: lineAtRune === lineCount-1,
        height: lineHeight,
        atRune: lineAtRune,
        avgLen: avgLineLength,
        width: rect.width,
    };
}

function firstTextNode(node) {
    
    for (const child of Array.from(node.childNodes)) {
        
        if (child.nodeType === Node.TEXT_NODE) {
            return child;
        }
        
        if (child.nodeType === Node.ELEMENT_NODE) {
            const tn = firstTextNode(child);
            if (tn) {
                return tn;
            }
        }
    }
}

function lastTextNode(node) {
    
    for (let i = node.childNodes.length-1; i >= 0; i--) {
        
        const child = node.childNodes[i];
        
        if (child.nodeType === Node.TEXT_NODE) {
            return child;
        }
        
        if (child.nodeType === Node.ELEMENT_NODE) {
            const tn = lastTextNode(child);
            if (tn) {
                return tn;
            }
        }
    }
}

// removeTextNode guarantees that it will also remove
// any parent nodes that are now empty as a result of
// its removal
function removeTextNode(tn) {
    
    let toRemove;
    let node = tn.parentNode;
    node.removeChild(tn);
    
    while (true) {
        
        if (node.classList.contains("input")) {
            break;
        }
        
        const tag = node.tagName.toLowerCase();
        const tags = editorParagraphs.concat(editorAtomics);
        if (contains(tags, tag)) {
            break;
        }
        
        if (node.textContent === "") {
            toRemove = node;
        }
        
        node = node.parentNode;
    }
    
    if (toRemove) {
        toRemove.parentNode.removeChild(toRemove);
    }
}
