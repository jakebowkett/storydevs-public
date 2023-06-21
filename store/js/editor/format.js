
/*=====================================================

    Formatting
    
    Our appraoch is to always expand formatting to
    the entire selection when it contains partial or
    no formatting of the kind the user is toggling.
    The only time it will disable the formatting is
    if the entire selection is formatted in the style
    the user has chosen.

=====================================================*/

function sameInlineFormatting(...nodes) {
    
    const node = nodes[0];
    const formats = formatList(node).inline;
    nodes = nodes.slice(1);
    
    for (const node of nodes) {
        
        const f = formatList(node).inline;
        
        if (f.length !== formats.length) {
            return false;
        }
        
        for (let i = 0; i < f.length; i++) {
            if (f[i] !== formats[i]) {
                return false;
            }
        }
    }
    
    return true;
}

function clearFormatMenu() {
    const tools = q(".tools", focusedEditor);
    for (const t of Array.from(tools.children)) {
        t.classList.remove("on");
    }
}

function updateFormatMenu(node, idx) {

    const ri = new RuneIterator(node, idx);
    let r = ri.current();
    
    const currentSpan = r.node.parentNode;
    const currentLink = currentSpan.dataset.link;
    
    if (selectionCollapsed()) {
        ri.prev();
        let prevRune = ri.current();
        if (prevRune !== null) {
            r = prevRune;
        }
    }
    
    const span = r.node.parentNode;
    const format = formatList(span);
    
    // Remove link style from inlines.
    for (let i = format.inline.length-1; i >= 0; i--) {
        if (format.inline[i] === "a") {
            format.inline.splice(i, 1);
            break;
        }
    }
    
    const tools = q(".tools", focusedEditor);
    for (const t of Array.from(tools.children)) {
        if (contains(ed.addFormat, t.dataset.format)) {
            t.classList.add("on");
            continue;
        }
        if (contains(ed.removeFormat, t.dataset.format)) {
            t.classList.remove("on");
            continue;
        }
        const inline = contains(format.inline, t.dataset.format);
        const block = contains(format.pLevel, t.dataset.format);
        if (inline || block) {
            t.classList.add("on");
            continue;
        }
        t.classList.remove("on");
    }
}

function formatList(node) {
    
    let inline = [];
    let pLevel = [];
    
    if (node.nodeType === Node.TEXT_NODE) {
        node = node.parentNode;
    }
    
    for (const cls of Array.from(node.classList)) {
        inline.push(cls);
    }
    
    while (true) {
        
        node = node.parentNode;
        
        if (node.classList.contains("input")) {
            break;
        }
        
        let tag = node.tagName.toLowerCase();

        if (tag === "p" || tag === "li") {
            continue;
        }
        
        if (tag === "blockquote") {
            tag = "bq";
        }
        
        pLevel.push(tag);
    }
    
    return {
        inline: inline,
        pLevel: pLevel,
    };
}

function updateAfterFormatButton() {
    const selection = orderedSelection();
    const start = selection.start;
    updateFormatMenu(start.node, start.idx);
    updateSelection();
}

function formatButton(e) {
    
    if (!focusedEditor) {
        return;
    }
    
    e.preventDefault();
    
    const btn = e.target;
    const format = btn.dataset.format;
    let keepNextFormat = false;
    
    switch (format) {
        
    case "b":
    case "i":
    case "u":
    // case "mono":
    // case "key":
        toggleInlineFormat(format);
        updateAfterFormatButton();
        keepNextFormat = true;
        break;
        
    case "a":
        addLink();
        break;
        
    case "h2":
        toggleAtomic("h2");
        break;
    case "bq":
        toggleAtomic("blockquote");
        break;
    
    case "ul":
    case "ol":
        toggleList(format === "ol");
        break;
        
    // case "code":
    }
    
    if (format !== "a") {
        updateSelection();
        dismissLinkEditor();
    }
    if (!keepNextFormat) {
        clearNextFormat();
    }
    updatePlaceholderText();
}

function selectedTextNodes(start, end, paras) {
    
    // If there's only one paragraph.
    if (start.p === end.p) {
        const ri = new RuneIterator(start.node);
        return ri.nodeRange(start.idx, end.idx);
    }
    
    let nodes = [];
    
    // First paragraph.
    let ri = new RuneIterator(start.node);
    let startRange = ri.nodeRange(start.idx, start.node.textContent.length);
    nodes = nodes.concat(startRange);
    
    // Middle paragraphs, if there are any.
    for (const p of paras.slice(1, -1)) {
        const ri = new RuneIterator(p);
        nodes = nodes.concat(ri.nodes);
    }
    
    // Last paragraph.
    ri = new RuneIterator(end.node);
    nodes = nodes.concat(ri.nodeRange(0, end.idx));
    
    return nodes;
}

function inlineFormatState(start, end, paras, format) {
    const nodes = selectedTextNodes(start, end, paras);
    for (const node of nodes) {
        const span = node.parentNode;
        if (!span.classList.contains(format)) {
            return true;
        }
    }
    return false;
}

function setNextFormat(format) {
    
    // Gather current inline formatting and data-
    // attributes at the current caret location.
    const focus = ed.focus;
    const ri = new RuneIterator(focus.node, focus.idx);
    if (focus.idx !== 0) {
        ri.prev();
    }
    const span = ri.current().node.parentNode;
    
    // Combine existing formats and addFormat.
    let af = ed.addFormat;
    let rf = ed.removeFormat;
    for (const cls of Array.from(span.classList)) {
        if (contains(af, cls)) {
            continue;
        }
        if (contains(rf, cls)) {
            continue;
        }
        af.push(cls);
    }
    
    // Remove format if it's already present.
    let seenFormat = false;
    for (let i = af.length-1; i >= 0; i--) {
        const f = af[i];
        if (f === format) {
            seenFormat = true;
            af.splice(i, 1);
            rf.push(format);
            break;
        }
    }
    
    // If we didn't see format we add it.
    if (!seenFormat) {
        af.push(format);
    }
}

function toggleInlineFormat(format, apply, remove) {
    
    if (selectionCollapsed()) {
        setNextFormat(format);
        return;
    }
    
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    const paragraphs = start.paragraphs.slice(start.p, end.p+1);
    
    // If data is supplied the "on" state is forced.
    // This is to prevent links from being toggled off.
    let on;
    if (apply) {
        on = true;
    }
    if (remove) {
        on = false;
    }
    if (!apply && !remove) {
        on = inlineFormatState(start, end, paragraphs, format);
    }
    
    for (let i = 0; i < paragraphs.length; i++) {
        
        const p = paragraphs[i];
        const tag = p.tagName.toLowerCase();
        if (!contains(editorParagraphs, tag) || tag === "code") {
            continue;
        }
        
        let startIdx;
        let endIdx;
        
        // Set appropriate ranges for current paragraph based
        // on whether it's at the start, middle, or end.
        if (paragraphs.length === 1) {
            startIdx = start.idx;
            endIdx = end.idx;
        }
        if (paragraphs.length > 1 && i === 0) {
            startIdx = start.idx;
            endIdx = p.textContent.length;
        }
        if (paragraphs.length > 1 && i > 0 && i < paragraphs.length-1) {
            startIdx = 0;
            endIdx = p.textContent.length;
        }
        if (paragraphs.length > 1 && i === paragraphs.length-1) {
            startIdx = 0;
            endIdx = end.idx;
        }
        
        const ri = new RuneIterator(p, startIdx);
        let r = ri.current();
        let tnStart = r.node;
        let offsetStart = r.offset;
        
        ri.seek(endIdx);
        r = ri.current();
        let tnEnd = r.node;
        let offsetEnd = r.offset;
        
        const nodes = ri.nodesBetween(tnStart, tnEnd);
        
        // If the end position is at the very start of a new
        // formatting span any toggling won't affect it.
        // So we adjust the cursor logical offset to the
        // end of the preceding formatting span.
        if (offsetEnd === 0) {
            if (nodes.length > 0) {
                tnEnd = nodes.pop();
            } else {
                tnEnd = tnStart;
            }
            offsetEnd = tnEnd.textContent.length;
        }
        
        if (tnStart === tnEnd) {
            
            const tc = tnStart.textContent;
            const afterSpan  = tnStart.parentNode;
            const selectSpan = afterSpan.cloneNode(false);
            const beforeSpan = afterSpan.cloneNode(false);
            
            afterSpan.textContent  = tc.slice(offsetEnd);
            selectSpan.textContent = tc.slice(offsetStart, offsetEnd);
            beforeSpan.textContent = tc.slice(0, offsetStart);
            
            if (on) {
                if (apply) apply(selectSpan);
                selectSpan.classList.add(format);
            } else {
                if (remove) remove(selectSpan);
                selectSpan.classList.remove(format);
            }
            
            p.insertBefore(selectSpan, afterSpan);
            p.insertBefore(beforeSpan, selectSpan);
            
            cleanInlineFormatting(p);
            
            continue;
        }
        
        // If the start offset is zero we update the
        // starting format span.
        const spanStart = tnStart.parentNode;
        if (offsetStart === 0) {
            if (on) {
                if (apply) apply(spanStart);
                spanStart.classList.add(format);
            } else {
                if (remove) remove(spanStart);
                spanStart.classList.remove(format);
            }
        } 
        
        // Otherwise we bisect the first format span.
        else {
            const tc = tnStart.textContent;
            tnStart.textContent = tc.slice(offsetStart);
            spanStartBefore = spanStart.cloneNode(false);
            spanStartBefore.textContent = tc.slice(0, offsetStart);
            p.insertBefore(spanStartBefore, spanStart);
            if (on) {
                if (apply) apply(spanStart);
                spanStart.classList.add(format);
            } else {
                if (remove) remove(spanStart);
                spanStart.classList.remove(format);
            }
        }
        
        // Handle middle format spans between start and end.
        for (const tn of nodes) {
            const span = tn.parentNode;
            if (on) {
                if (apply) apply(span);
                span.classList.add(format);
            } else {
                if (remove) remove(span);
                span.classList.remove(format);
            }
        }
        
        // If the offset is zero we update the
        // end format span.
        const spanEnd = tnEnd.parentNode;
        if (offsetEnd === 0) {
            if (on) {
                if (apply) apply(spanEnd);
                spanEnd.classList.add(format);
            } else {
                if (remove) remove(spanEnd);
                spanEnd.classList.remove(format);
            }
        }
        
        // Otherwise we bisect the last format span.
        else {
            const tc = tnEnd.textContent;
            tnEnd.textContent = tc.slice(offsetEnd);
            spanEndBefore = spanEnd.cloneNode(false);
            spanEndBefore.textContent = tc.slice(0, offsetEnd);
            p.insertBefore(spanEndBefore, spanEnd);
            if (on) {
                if (apply) apply(spanEndBefore);
                spanEndBefore.classList.add(format);
            } else {
                if (remove) remove(spanEndBefore);
                spanEndBefore.classList.remove(format);
            }
        }
        
        cleanInlineFormatting(p);
    }
}

function cleanInlineFormatting(p) {
    
    // Remove empty spans.
    for (let i = p.children.length-1; i >= 0; i--) {
    
        const span = p.children[i];
    
        if (span.textContent.length === 0) {
            p.removeChild(span);
            continue;
        }
    }
    
    // Merge spans with same formatting.
    for (let i = p.children.length-1; i >= 0; i--) {
    
        const span = p.children[i];
        
        if (i === 0) {
            continue;
        }
        
        const prevSpan = p.children[i-1];
        if (!sameClasses(span, prevSpan)) {
            continue;
        }
        
        // Some inline formats may have additional
        // data associated with them. Links have urls
        // for example. The datasets of two formatting
        // spans should also match before merging them
        // otherwise links with different URLs could
        // accidentally be merged.
        if (!sameDataset(span, prevSpan)) {
            continue;
        }
        
        prevSpan.textContent += span.textContent;
        p.removeChild(span);
    }
}

function sameDataset(e1, e2) {
    if (e1.dataset.length !== e2.dataset.length) {
        return false;
    }
    for (const prop in e1.dataset) {
        if (!(prop in e2.dataset)) {
            return false;
        }
        if (e1.dataset[prop] !== e2.dataset[prop]) {
            return false;
        }
    }
    return true;
}

function sameClasses(e1, e2) {
    if (e1.classList.length !== e2.classList.length) {
        return false;
    }
    for (const cls of Array.from(e1.classList)) {
        if (!e2.classList.contains(cls)) {
            return false;
        }
    }
    return true;
}

function blockQuoteToggleState(paras) {
    for (const p of paras) {
        if (!findAncestor("blockquote", p)) {
            return true;
        }
    }
    return false;
}

function toggleAtomic(tag) {
    
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    const paras = start.paragraphs.slice(start.p, end.p+1);
    const on = atomicToggleState(paras, tag);
    
    if (!on) {
        switchParaTags(start, end, paras, "p");
        updateFormatMenu(start.node, start.idx);
        updateSelection();
        return;
    }
    
    listOff(start, end, paras);
    switchParaTags(start, end, paras, tag);
    
    // When applying the heading format we remove
    // any inline styling.
    for (const h of paras) {
        const tc = h.textContent;
        
        // Notice we stop before the first span.
        for (let i = h.children.length-1; i > 0; i--) {
            removeNode(h.children[i]);
        }
        
        h.children[0].textContent = tc;
        h.children[0].removeAttribute("class");
        h.children[0].removeAttribute("data-link");
    }
    
    updateFormatMenu(start.node, start.idx);
    updateSelection();
}

function switchParaTags(start, end, paras, tag) {
    
    for (let i = 0; i < paras.length; i++) {
        
        const oldP = paras[i];
        const newP = document.createElement(tag);
        for (let i = oldP.children.length-1; i >= 0; i--) {
            const child = oldP.children[i];
            prependChild(child, newP);
        }
        replaceNode(newP, oldP);
        
        // Update references.
        paras[i] = newP;
        start.paragraphs[start.p+i] = newP;
        end.paragraphs[start.p+i] = newP;
        if (i === 0) {
            start.node = newP;
        }
        if (i === paras.length-1) {
            end.node = newP;
        }
    }
}

function atomicToggleState(paras, tag) {
    for (const p of paras) {
        if (tagName(p) !== tag) {
            return true;
        }
    }
    return false;
}

function toggleList(ordered) {
    
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    const paras = start.paragraphs.slice(start.p, end.p+1);
    const on = listToggleState(ordered, paras);
    
    if (!on) {
        listOff(start, end, paras);
        return;
    }
    
    const list = document.createElement(ordered ? "ol" : "ul");
    const tag = tagName(paras[0].parentNode);
    const inList = contains(editorLists, tag);
    
    if (!inList) {
        insertBefore(list, paras[0]);
        parasToList(start, end, paras, list);
        return;
    }
    
    listOff(start, end, paras);
    insertAfter(list, paras[0]);
    parasToList(start, end, paras, list);
    return;
}

function listToggleState(ordered, paras) {
    for (const p of paras) {
        if (tagName(p) !== "li") {
            return true;
        }
        const tag = tagName(p.parentNode);
        if (differentListType(ordered, tag)) {
            return true;
        }
    }
    return false;
}

function parasToList(start, end, paras, list) {
    
    // We need the iteration count for updating refs
    // hence we don't use a for ... of style loop.
    for (let i = 0; i < paras.length; i++) {
        
        const p = paras[i];
        const tag = tagName(p);
        
        // Move paragraph if it's already a list item.
        if (tag === "li") {
            
            const parent = p.parentNode;
            list.appendChild(p);
            
            // Remove former list if it's now empty.
            if (elemEmpty(parent)) {
                removeNode(parent);
            }
            
            continue;
        }
        
        // Otherwise transfer paragraph's contents to a new list item.
        const item = document.createElement("li");
        for (let i = p.children.length-1; i >= 0; i--) {
            const child = p.children[i];
            prependChild(child, item);
            
        }
        
        list.appendChild(item);
        removeNode(p);
        
        // Update paragraph reference.
        start.paragraphs[start.p+i] = item;
        end.paragraphs[start.p+i] = item;
        if (i === 0) {
            start.node = item;
        }
        if (i === paras.length-1) {
            end.node = item;
        }
    }
    
    // Combine lists of the same type.
    combineLists();
}

function listOff(start, end, paras) {
    
    // If first paragraph isn't in a list we
    // insert all the paragraphs after it.
    const firstP = paras[0];
    const tag = tagName(firstP.parentNode);
    if (!contains(editorLists, tag)) {
        
        // Create a reference point before which we
        // will insert paragraphs.
        const refNode = document.createElement("div");
        insertAfter(refNode, firstP);
        parasFromList(start, end, paras, refNode);
        removeNode(refNode);
        return;
    }
    
    // Otherwise we bisect the list it's in, insert
    // the latter half after the former and insert
    // paras after the former but before the latter.
    const listAfter = firstP.parentNode;
    const listBefore = document.createElement(tag);
    const itemsBefore = [];
    for (const item of Array.from(listAfter.children)) {
        if (item === paras[0]) {
            break;
        }
        itemsBefore.push(item);
    }
    for (const item of itemsBefore) {
        listBefore.appendChild(item);
    }
    insertBefore(listBefore, listAfter);
    parasFromList(start, end, paras, listAfter);
}

// Note: we work with the assumption that some of the
// supplied paras may not actually belong to lists.
function parasFromList(start, end, paras, refNode) {
    
    for (let i = 0; i < paras.length; i++) {
        
        const item = paras[i];
        const p = document.createElement("p");
        for (let i = item.children.length-1; i >= 0; i--) {
            const child = item.children[i];
            prependChild(child, p);
        }
        removeNode(item);
        insertBefore(p, refNode);
        
        // Update references.
        paras[i] = p;
        start.paragraphs[start.p+i] = p;
        end.paragraphs[start.p+i] = p;
        if (i === 0) {
            start.node = p;
        }
        if (i === paras.length-1) {
            end.node = p;
        }
    }
    
    removeEmptyLists();
}

function removeEmptyLists() {
    
    function recurse(node) {
        
        for (let i = node.children.length-1; i >= 0; i--) {
        
            const child = node.children[i];
            const tag = child.tagName.toLowerCase();
            
            // Recurse into containers.
            if (contains(editorContainers, tag)) {
                recurse(child);
                continue;
            }
            
            if (!contains(editorLists, tag)) {
                continue;
            }
            
            if (child.children.length > 0) {
                continue;
            }
            
            node.removeChild(child);
        }
    }
    
    recurse(q(".input", focusedEditor));
}

function differentListType(ordered, tag) {
    if (ordered && tag === "ul") {
        return true;
    }
    if (!ordered && tag === "ol") {
        return true;
    }
    return false;
}

function combineLists() {
    
    function recurse(node) {
        
        let prevElem;
        let prevTag;
        
        for (let i = node.children.length-1; i >= 0; i--) {
            
            const child = node.children[i];
            const tag = tagName(child);
            
            // Recurse into containers.
            if (contains(editorContainers, tag)) {
                recurse(child);
                prevElem = child;
                prevTag = tag;
                continue;
            }
            
            // Continue if current elem is not list or if it's
            // not the same type as the previous list.
            if (!contains(editorLists, tag) || tag !== prevTag) {
                prevElem = child;
                prevTag = tag;
                continue;
            }
            
            // We have a list that's the same type as the previous list.
            const items = [];
            for (const item of Array.from(prevElem.children)) {
                items.push(item);
            }
            for (const item of items) {
                child.appendChild(item);
            }
            removeNode(prevElem);
            prevElem = child;
            prevTag = tag;
        }
    }
    
    recurse(q(".input", focusedEditor));
}