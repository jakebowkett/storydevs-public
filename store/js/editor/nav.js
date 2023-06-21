
function navVertical(up, expand) {
    
    const focus = ed.focus;
    
    if (up) {
        
        // Can't go up.
        if (focus.p === 0 && focus.line === 0) {
            toEditorBoundary(true, expand);
            return;
        }
        
        // Go to preceding paragraph.
        if (focus.line === 0) {
            
            const newP = focus.paragraphs[focus.p-1];
            const pRect = newP.getBoundingClientRect();
            const line = lineFromPoint(newP, pRect.bottom);
            const lineStartIdx = line.avgLen * line.atPoint;
            const maxIdx = newP.textContent.length-1;
            const xOffset = focus.xMem - pRect.left;
            const xPos = clamp(xOffset / pRect.width, 0, 1);
            const approxIdx = clamp(Math.round(lineStartIdx + (xPos * line.avgLen)), 0, maxIdx);
            
            caret = caretFromPoint(newP, focus.xMem, line.top, approxIdx);
            
            focus.node = newP;
            focus.pEmpty = newP.textContent === edEmptyChar;
            focus.p--;
            focus.lastP = false;
            focus.line = line.atPoint;
            focus.lastLine = line.last;
            focus.idx = focus.pEmpty ? 0 : caret.idx;
            focus.caretX = caret.x;
            focus.caretY = line.top;
            focus.caretHeight = line.height;
            
            updateCaretAndSelection(expand, caret.x, focus.caretY, focus.caretHeight);
        
            return;
        }
        
        // Go to previous line within current paragraph.
        const p = focus.node;
        const pRect = p.getBoundingClientRect();
        const lineTop = focus.caretY - focus.caretHeight;
        const lineCount = pRect.height / focus.caretHeight;
        const avgLineLength = Math.round((p.textContent.length-1) / lineCount);
        const lineStartIdx = avgLineLength * focus.line;
        const maxIdx = p.textContent.length-1;
        const xOffset = focus.xMem - pRect.left;
        const xPos = clamp(xOffset / pRect.width, 0, 1);
        const approxIdx = clamp(Math.round(lineStartIdx + (xPos * avgLineLength)), 0, maxIdx);
        
        caret = caretFromPoint(p, focus.xMem, lineTop, approxIdx);
        
        focus.line -= 1;
        focus.lastLine = false;
        focus.idx = caret.idx;
        focus.caretX = caret.x;
        focus.caretY -= focus.caretHeight;
        
        updateCaretAndSelection(expand, caret.x, focus.caretY, focus.caretHeight);
        
        return;
    }
    
    // Can't go down.
    if (focus.lastP && focus.lastLine) {
        toEditorBoundary(false, expand);
        return;
    }
    
    // Go to next paragraph.
    if (focus.lastLine) {
        
        const newP = focus.paragraphs[focus.p+1];
        const pRect = newP.getBoundingClientRect();
        const line = lineFromPoint(newP, pRect.top);
        const lineStartIdx = line.avgLen * line.atPoint;
        const maxIdx = newP.textContent.length-1;
        const xOffset = focus.xMem - pRect.left;
        const xPos = clamp(xOffset / pRect.width, 0, 1);
        const approxIdx = clamp(Math.round(lineStartIdx + (xPos * line.avgLen)), 0, maxIdx);
        
        caret = caretFromPoint(newP, focus.xMem, line.top, approxIdx);
        
        focus.node = newP;
        focus.pEmpty = newP.textContent === edEmptyChar;
        focus.p++;
        focus.lastP = focus.p === focus.paragraphs.length-1;
        focus.line = line.atPoint;
        focus.lastLine = line.last;
        focus.idx = focus.pEmpty ? 0 : caret.idx;
        focus.caretX = caret.x;
        focus.caretY = line.top;
        focus.caretHeight = line.height;
        
        updateCaretAndSelection(expand, caret.x, focus.caretY, focus.caretHeight);
    
        return;
    }
    
    // Go to next line within current paragraph.
    const p = focus.node;
    const pRect = p.getBoundingClientRect();
    const lineTop = focus.caretY + focus.caretHeight;
    const lineCount = pRect.height / focus.caretHeight;
    const avgLineLength = Math.round((p.textContent.length-1) / lineCount);
    const lineStartIdx = avgLineLength * (focus.line+1);
    const maxIdx = p.textContent.length-1;
    const xOffset = focus.xMem - pRect.left;
    const xPos = clamp(xOffset / pRect.width, 0, 1);
    const approxIdx = clamp(Math.round(lineStartIdx + (xPos * avgLineLength)), 0, maxIdx);
    
    caret = caretFromPoint(p, focus.xMem, lineTop, approxIdx);
    
    focus.line += 1;
    focus.lastLine = focus.line === lineCount-1;
    focus.idx = caret.idx;
    focus.caretX = caret.x;
    focus.caretY += focus.caretHeight;
    
    updateCaretAndSelection(expand, caret.x, focus.caretY, focus.caretHeight);
}

function toEditorBoundary(start, expand) {
    
    const focus = ed.focus;
    
    if (start) {
        
        const p = focus.paragraphs[0];
        const lineHeight = editorLineHeight(p);
        const rect = p.getBoundingClientRect();
        const lineCount = rect.height / lineHeight;
        
        focus.node = p;
        focus.pEmpty = p.textContent === edEmptyChar;
        focus.p = 0;
        focus.lastP = focus.paragraphs.length === 1;
        focus.line = 0;
        focus.lastLine = lineCount === 1;
        focus.idx = 0;
        focus.caretX = rect.left;
        focus.caretY = rect.top;
        focus.caretHeight = lineHeight;
        
        updateCaretAndSelection(expand, rect.left, rect.top, lineHeight);
        
        return;
    }
    
    const p = focus.paragraphs[focus.paragraphs.length-1];
    const ri = new RuneIterator(p, p.textContent.length);
    const r = ri.current();
    const range = document.createRange();

    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    
    const rect = range.getBoundingClientRect();
    const line = lineFromRune(p, rect);
    
    focus.node = p;
    focus.pEmpty = p.textContent === edEmptyChar;
    focus.p = focus.paragraphs.length-1;
    focus.lastP = true;
    focus.line = line.atRune;
    focus.lastLine = line.last;
    focus.idx = focus.pEmpty ? 0 : p.textContent.length;
    focus.caretX = rect.right;
    focus.caretY = line.top;
    focus.caretHeight = line.height;
    
    updateCaretAndSelection(expand, rect.right, line.top, line.height);
}

function navHorizontal(backward, expand) {
    
    const focus = ed.focus;
    const range = document.createRange();

    // If there's a selection we don't advance the
    // caret beyond the start or end boundary.
    if (!expand && !selectionCollapsed()) {
        const selection = orderedSelection();
        const start = selection.start;
        const end = selection.end;
        if (backward) {
            Object.assign(end, start);
        } else {
            Object.assign(start, end);
        }
        placeCaretAt(start.caretX, start.caretY, start.caretHeight);
        clearEditorCanvas();
        return;
    }
    
    if (backward) {
        
        // Can't go back.
        if (focus.p === 0 && focus.idx === 0) {
            updateCaretAndSelection(expand, focus.caretX, focus.caretY, focus.caretHeight);
            return;
        }
        
        // Go to end of preceding paragraph.
        if (focus.idx === 0) {
            
            const newP = focus.paragraphs[focus.p-1];
            
            // Handle empty paragraph special case. We can't
            // actually use empty paragraphs because Chrome
            // deletes empty text nodes. We need a text node
            // inside our paragraph elements otherwise their
            // height is off (also the current code cannot
            // handle paragraph elements with no text nodes).
            // Therefore empty paragraphs are represented by
            // an element containing a single zero-width space.
            let idx;
            let pEmpty;
            if (newP.textContent === edEmptyChar) {
                pEmpty = true;
                idx = 0;
            } else {
                pEmpty = false;
                idx = newP.textContent.length;
            }
            
            const ri = new RuneIterator(newP, idx);
            const r = ri.current();
            
            range.setStart(r.node, r.offset);
            range.setEnd(r.node, r.offset + r.len);
            
            const rect = range.getBoundingClientRect();
            const line = lineFromRune(newP, rect);
            
            focus.node = newP;
            focus.pEmpty = pEmpty;
            focus.p = focus.p-1;
            focus.lastP = focus.p === focus.paragraphs.length-1;
            focus.line = line.atRune;
            focus.lastLine = line.last;
            focus.idx = idx;
            focus.caretX = rect.right;
            focus.caretY = line.top;
            focus.caretHeight = line.height;
            
            updateCaretAndSelection(expand, focus.caretX, focus.caretY, focus.caretHeight);
            
            return;
        }
        
        // Move back within current paragraph.
        const ri = new RuneIterator(focus.node, focus.idx);
        ri.prev();
        const r = ri.current();
        
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset + r.len);
        
        const rect = range.getBoundingClientRect();
        const line = lineFromRune(focus.node, rect);
        
        focus.line = line.atRune;
        focus.lastLine = line.last;
        focus.idx = r.overall;
        focus.caretX = rect.left;
        focus.caretY = line.top;
        focus.caretHeight = line.height;
    
        updateCaretAndSelection(expand, focus.caretX, focus.caretY, focus.caretHeight);
    
        return;
    }

    // Can't go forward.
    if (focus.lastP && focus.idx === focus.node.textContent.length) {
        updateCaretAndSelection(expand, focus.caretX, focus.caretY, focus.caretHeight);
        return;
    }
    
    // Move to start of following paragraph.
    if (focus.idx === focus.node.textContent.length) {
        moveToNextParagraph();
        return;
    }

    // Move forward within current paragraph.
    const ri = new RuneIterator(focus.node, focus.idx);
    ri.next();
    const r = ri.current();
    
    if (focus.node.textContent === edEmptyChar) {
        moveToNextParagraph();
        return;
    }
    
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    
    const rect = range.getBoundingClientRect();
    const line = lineFromRune(focus.node, rect);
    
    focus.line = line.atRune;
    focus.lastLine = line.last;
    focus.idx = r.overall;
    focus.caretX = rect.left;
    focus.caretY = line.top;
    focus.caretHeight = line.height;

    updateCaretAndSelection(expand, focus.caretX, focus.caretY, focus.caretHeight);
    
    function moveToNextParagraph() {
        
        if (focus.lastP) {
            return;
        }
        
        const newP = focus.paragraphs[focus.p+1];
        const ri = new RuneIterator(newP, 0);
        const r = ri.current();
        
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset + r.len);
        
        const rect = range.getBoundingClientRect();
        const line = lineFromRune(newP, rect);
        
        focus.node = newP;
        focus.pEmpty = newP.textContent === edEmptyChar;
        focus.p = focus.p+1;
        focus.lastP = focus.p === focus.paragraphs.length-1;
        focus.line = line.atRune;
        focus.lastLine = line.last;
        focus.idx = 0;
        focus.caretX = rect.left;
        focus.caretY = line.top;
        focus.caretHeight = line.height;
        
        updateCaretAndSelection(expand, focus.caretX, focus.caretY, focus.caretHeight);
    }
}

function toWordBoundary(start, expand) {
    
    const focus = ed.focus;
    
    if (focus.idx === 0 && start) {
        navHorizontal(true, expand);
        return;
    } 
    if (focus.idx === focus.node.textContent.length && !start) {
        navHorizontal(false, expand);
        return;
    }
    if (focus.pEmpty) {
        navHorizontal(start, expand);
        return;
    }
    
    const ri = new RuneIterator(focus.node, focus.idx);
    const range = document.createRange();
    
    let idx;
    let caretX;
    let prevTop = focus.caretY;
    let line = focus.line;
    let lineChanged = false;
    
    // Effectively the longest word allowed, measured
    // in code points.
    let maxIterations = 256;
    let loopCount = 0;
    
    while (true) {
        
        loopCount++;
        if (loopCount > maxIterations) {
            log("breaking infinite loop");
            break;
        }
        
        if (start) {
            ri.prev();
        } else {
            ri.next();
        }
        
        const r = ri.current();
        
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset + r.len);
        
        const rect = range.getBoundingClientRect();
        const diff = prevTop - rect.top;
        
        if (diff < -editorLineThreshold) {
            line++;
            lineChanged = true;
        }
        if (diff > editorLineThreshold) {
            line--;
            lineChanged = true;
        }
        prevTop = rect.top;
        
        if (r.overall === 0) {
            idx = 0;
            caretX = rect.left;
            break;
        }
        if (r.overall === focus.node.textContent.length) {
            idx = focus.node.textContent.length;
            caretX = rect.right;
            break;
        }
        
        if (loopCount === 1 && !lineChanged) {
            continue;
        }
        
        if (contains(editorSkipPoints, r.rune) || r.rune.match(/\s/)) {
            if (start) {
                idx = r.overall + 1;
                caretX = rect.right;
            } else {
                idx = r.overall;
                caretX = rect.left;
            }
            break;
        }
    }
    
    focus.idx = idx;
    focus.line = line;
    focus.lastLine = line === (focus.node.clientHeight / focus.caretHeight) - 1;
    focus.caretX = caretX;
    focus.caretY = focus.node.getBoundingClientRect().top + (focus.caretHeight * line);

    updateCaretAndSelection(expand, focus.caretX, focus.caretY, focus.caretHeight);
}

function toLineBoundary(start, expand) {
    
    const focus = ed.focus;
    const p = focus.node;
    const pRect = p.getBoundingClientRect();
    
    const lineCount = p.clientHeight / focus.caretHeight;
    const avgLineLength = Math.round((p.textContent.length-1) / lineCount);
    const lineStartIdx = Math.round(avgLineLength * focus.line);
    
    let xCoord;
    let approxIdx;
    if (start) {
        xCoord = pRect.left;
        approxIdx = lineStartIdx;
    } else {
        xCoord = pRect.right;
        approxIdx = lineStartIdx + avgLineLength;
    }
    
    const caret = caretFromPoint(p, xCoord, focus.caretY, approxIdx);
    
    focus.idx = caret.idx;
    focus.caretX = caret.x;
    
    updateCaretAndSelection(expand, caret.x, focus.caretY, focus.caretHeight);
}

function updateCaretAndSelection(expand, caretX, caretY, caretHeight) {
    
    placeCaretAt(caretX, caretY, caretHeight);
    
    if (expand) {
        drawSelection();
        return;
    }
    
    Object.assign(ed.anchor, ed.focus);
    clearEditorCanvas();
}