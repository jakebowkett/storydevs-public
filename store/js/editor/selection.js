
function selectWord() {
    toWordBoundary(true);
    toWordBoundary(false, true);
}

function selectParagraph() {
    
    const focus = ed.focus;
    const anchor = ed.anchor;
    const p = focus.node;
    const ri = new RuneIterator(p, p.textContent.length);
    const r = ri.current();
    
    const range = document.createRange();
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    const rect = range.getBoundingClientRect();
    const pRect = p.getBoundingClientRect();
    const line = lineFromRune(p, rect);
    
    anchor.line = 0;
    anchor.lastLine = line.count === 1;
    anchor.idx = 0;
    anchor.caretX = pRect.left;
    anchor.caretY = pRect.top;
    
    focus.line = line.atRune;
    focus.lastLine = true;
    focus.idx = p.textContent.length;
    focus.caretX = rect.right;
    focus.caretY = line.top;
    
    placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
    drawSelection();
}

function selectAll() {
    
    const anchor = ed.anchor;
    const focus = ed.focus;
    
    let p = anchor.paragraphs[0];
    let rect = p.getBoundingClientRect();
    let lineHeight = editorLineHeight(p);
    
    anchor.node = p;
    anchor.p = 0;
    anchor.pEmpty = p.textContent === edEmptyChar;
    anchor.lastP = anchor.paragraphs.length-1 === 0;
    anchor.line = 0;
    anchor.lastLine = rect.height === lineHeight;
    anchor.idx = 0;
    anchor.caretX = rect.left;
    anchor.caretY = rect.top;
    anchor.caretHeight = lineHeight;
    
    p = focus.paragraphs[focus.paragraphs.length-1];
    lineHeight = editorLineHeight(p);
    lineCount = rect.clientHeight / lineHeight;
    const ri = new RuneIterator(p, p.textContent.length);
    const r = ri.current();
    const range = document.createRange();
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    rect = p.getBoundingClientRect();
    
    focus.node = p;
    focus.p = focus.paragraphs.length-1;
    focus.pEmpty = focus.node.textContent === edEmptyChar;
    focus.lastP = focus.paragraphs.length-1 === 0;
    focus.line = lineCount;
    focus.lastLine = rect.clientHeight === lineHeight;
    focus.idx = p.textContent.length;
    focus.caretX = rect.right;
    focus.caretY = rect.top;
    focus.caretHeight = lineHeight;
    
    drawSelection();
}

function orderedSelection() {
    
    const anchor = ed.anchor;
    const focus = ed.focus;
    
    if (focus.p === undefined) {
        return {
            start: anchor,
        };
    }
    
    if (anchor.p > focus.p) {
        return {
            start: focus,
            end: anchor,
            switched: true,
        };
    }
    if (anchor.p < focus.p) {
        return {
            start: anchor,
            end: focus,
        };
    }
    
    // If paragraphs are the same...
    if (anchor.idx > focus.idx) {
        return {
            start: focus,
            end: anchor,
            switched: true,
        };
    }
    return {
        start: anchor,
        end: focus,
    };
}

function clearEditorCanvas() {
    const canvas = q("canvas", focusedEditor);
    const ctx = canvas.getContext("2d");
    ctx.clearRect(0, 0, canvas.width, canvas.height);
}

function updateSelection(scrollInner) {
    
    // There is no selection without a focused editor.
    if (!focusedEditor) {
        return;
    }
    
    // If focusedEditor is not a child of the
    // element being scrolled we return. If
    // there is no scrollInner we ignore this.
    if (scrollInner) {
        let node = focusedEditor;
        let isChild = false;
        while (true) {
            
            if (node === scrollInner) {
                isChild = true;
                break;
            }
            
            node = node.parentNode;
        }
        if (!isChild) {
            return;
        }
    }
    
    ed.anchor = caretFromPos(ed.anchor);
    ed.focus = caretFromPos(ed.focus);
    const focus = ed.focus;
        
    drawSelection();
    placeCaretAt(focus.caretX, focus.caretY, focus.caretHeight);
}

function selectionCollapsed() {
    const anchor = ed.anchor;
    const focus = ed.focus;
    const sameP = anchor.p === focus.p;
    const sameLine = anchor.line === focus.line;
    const sameIdx = anchor.idx === focus.idx;
    if (sameP && sameLine && sameIdx) {
        return true;
    }
    return false;
}

function drawSelection() {
    
    clearEditorCanvas();
    
    const anchor = ed.anchor;
    const focus = ed.focus;
    
    if (selectionCollapsed()) {
        return;
    }
    
    const canvas = q("canvas", focusedEditor);
    const cRect = canvas.getBoundingClientRect();
    const offsetTop = cRect.top;
    const offsetLeft = cRect.left;
    
    const selection = orderedSelection();
    const start = selection.start;
    const end = selection.end;
    
    const ctx = canvas.getContext("2d");
    
    // Single paragraph.
    if (start.p === end.p) {
        
        // Within one line.
        if (start.line === end.line) {
            
            ctx.beginPath();
            
            moveTo(ctx, start.caretX, start.caretY);
            lineTo(ctx, end.caretX, end.caretY);
            lineTo(ctx, end.caretX, end.caretY + end.caretHeight);
            lineTo(ctx, start.caretX, start.caretY + start.caretHeight);
            lineTo(ctx, start.caretX, start.caretY);
            
            ctx.fill();
            ctx.closePath();
            
            return;
        }
        
        // Two separate lines.
        const neighbouringLines = Math.abs(start.line - end.line) === 1;
        if (neighbouringLines && end.caretX < start.caretX) {
            
            const pEnd = start.node.getBoundingClientRect().right;
            const pStart = end.node.getBoundingClientRect().left;
            
            ctx.beginPath();
            
            moveTo(ctx, start.caretX, start.caretY);
            lineTo(ctx, pEnd, start.caretY);
            lineTo(ctx, pEnd, start.caretY + start.caretHeight);
            lineTo(ctx, start.caretX, start.caretY + start.caretHeight);
            lineTo(ctx, start.caretX, start.caretY);
            
            moveTo(ctx, pStart, end.caretY);
            lineTo(ctx, end.caretX, end.caretY);
            lineTo(ctx, end.caretX, end.caretY + end.caretHeight);
            lineTo(ctx, pStart, end.caretY + end.caretHeight);
            lineTo(ctx, pStart, end.caretY);
            
            ctx.fill();
            ctx.closePath();
            
            return;
        }
        
        // Contiguous lines.
        const pEnd = start.node.getBoundingClientRect().right;
        const pStart = end.node.getBoundingClientRect().left;
        
        ctx.beginPath();
        
        moveTo(ctx, start.caretX, start.caretY);
        lineTo(ctx, pEnd, start.caretY);
        lineTo(ctx, pEnd, end.caretY);
        lineTo(ctx, end.caretX, end.caretY);
        lineTo(ctx, end.caretX, end.caretY + end.caretHeight);
        lineTo(ctx, pStart, end.caretY + end.caretHeight);
        lineTo(ctx, pStart, start.caretY + start.caretHeight);
        lineTo(ctx, start.caretX, start.caretY + start.caretHeight);
        lineTo(ctx, start.caretX, start.caretY);
        
        ctx.fill();
        ctx.closePath();    
        
        return;
    }
    
    // Multiple paragraphs.
    const middleParagraphs = end.paragraphs.slice(start.p+1, end.p);
    const startRect = start.node.getBoundingClientRect();
    const endRect = end.node.getBoundingClientRect();
    ctx.beginPath();
    
    // Start paragraph.
    if (start.lastLine) { // single line
        let right;
        if (selection.switched && start.pEmpty) {
            right = start.caretX;
        } else {
            right = startRect.right;
        }
        moveTo(ctx, start.caretX, start.caretY);
        lineTo(ctx, right, start.caretY);
        lineTo(ctx, right, start.caretY + start.caretHeight);
        lineTo(ctx, start.caretX, start.caretY + start.caretHeight);
        lineTo(ctx, start.caretX, start.caretY);
        
    } else { // multi line
        moveTo(ctx, start.caretX, start.caretY);
        lineTo(ctx, startRect.right, start.caretY);
        lineTo(ctx, startRect.right, startRect.bottom);
        lineTo(ctx, startRect.left, startRect.bottom);
        lineTo(ctx, startRect.left, start.caretY + start.caretHeight);
        lineTo(ctx, start.caretX, start.caretY + start.caretHeight);
        lineTo(ctx, start.caretX, start.caretY);
    }
    
    // Middle paragraphs.
    for (let p of middleParagraphs) {
        
        const rect = p.getBoundingClientRect();
        
        let right;
        if (p.textContent.length === 0) {
            // right = rect.left + 10;
            right = rect.right;
        } else {
            right = rect.right;
        }
        
        moveTo(ctx, rect.left, rect.top);
        lineTo(ctx, right, rect.top);
        lineTo(ctx, right, rect.bottom);
        lineTo(ctx, rect.left, rect.bottom);
        lineTo(ctx, rect.left, rect.top);
    }
    
    // End paragraph.
    if (end.line === 0) { // single line
        if (selection.switched && end.caretX === endRect.left) {
            end.caretX += 10;
        }
        moveTo(ctx, end.caretX, end.caretY);
        lineTo(ctx, end.caretX, end.caretY + end.caretHeight);
        lineTo(ctx, endRect.left, end.caretY + end.caretHeight);
        lineTo(ctx, endRect.left, end.caretY);
        lineTo(ctx, end.caretX, end.caretY);
    } else { // multi line
        moveTo(ctx, end.caretX, end.caretY);
        lineTo(ctx, end.caretX, end.caretY + end.caretHeight);
        lineTo(ctx, endRect.left, end.caretY + end.caretHeight);
        lineTo(ctx, endRect.left, endRect.top);
        lineTo(ctx, endRect.right, endRect.top);
        lineTo(ctx, endRect.right, end.caretY);
        lineTo(ctx, end.caretX, end.caretY);
    }
    
    ctx.fill();
    ctx.closePath();
        
    function moveTo(ctx, x, y) {
        ctx.moveTo(
            Math.round(x - offsetLeft) + 0.5,
            Math.round(y - offsetTop)  + 0.5
        );
    }

    function lineTo(ctx, x, y) {
        ctx.lineTo(
            Math.round(x - offsetLeft) + 0.5,
            Math.round(y - offsetTop)  + 0.5
        );
    }
}

function paragraphFromPoint(node, y) {
    
    const paras = [];
    lineariseParagraphs(node, paras);
    
    let closestNode;
    let closestIdx;
    let smallestDelta;
    
    for (let i = 0; i < paras.length; i++) {
        
        const p = paras[i];
        const rect = p.getBoundingClientRect();
        
        // Inside the paragraph.
        if (y > rect.top && y < rect.bottom) {
            closestNode = p;
            closestIdx = i;
            break;
        }
        
        const deltaTop = Math.abs(rect.top - y);
        const deltaBottom = Math.abs(rect.bottom - y);
        const deltaY = deltaTop < deltaBottom ? deltaTop : deltaBottom;
        
        // We've started getting further away.
        if (smallestDelta !== undefined && deltaY >= smallestDelta) {
            break;
        }
        
        // Otherwise we're getting closer.
        closestNode = p;
        closestIdx = i;
        smallestDelta = deltaY;
    }
    
    return {
        node: closestNode,
        idx: closestIdx,
        paragraphs: paras,
    };
}

function lineariseParagraphs(node, paras) {
    
    for (const child of Array.from(node.children)) {
        
        const tag = child.tagName.toLowerCase();
        const isIgnore = contains(editorIgnore, tag);
        
        if (isIgnore) {
            continue;
        }
        
        const isContainer = contains(editorContainers, tag);
        const isList = contains(editorLists, tag);
        
        if (isContainer || isList) {
            lineariseParagraphs(child, paras);
            continue;
        }
        
        paras.push(child);
    }
}

function caretFromPos(pos) {
    let coords = newCaretCoords(pos, pos.node);
    pos.caretX = coords.caretX ? coords.caretX : pos.caretX;
    pos.caretY = coords.caretY;
    pos.caretHeight = coords.caretHeight;
    pos.line = coords.targetLine;
    pos.lastLine = coords.totalLines-1 === coords.targetLine;
    return pos;
}


function nullRect(r) {
    if (
        r.top === 0 &&
        r.bottom === 0 &&
        r.left === 0 &&
        r.right === 0 &&
        r.width === 0 &&
        r.height === 0 &&
        r.x === 0 &&
        r.y === 0
    ) {
        return true;
    }
    return false;
}

/*
TODO: checking the window width is flawed because the *editor*
width could've changed without adjusting the window (e.g. switching
column layouts)
*/
function newCaretCoords(target, node) {
    
    const ri = new RuneIterator(node, target.idx);
    const range = document.createRange();
    const lineHeight = editorLineHeight(node);
    const nodeRect = node.getBoundingClientRect();
    
    let r = ri.current();
    
    range.setStart(r.node, r.offset);
    range.setEnd(r.node, r.offset + r.len);
    let rect = range.getBoundingClientRect();
    let x = rect.left;
    
    // In Safari the bounding rect for a range from
    // len to len is zeroed-out which messes up calculations
    // below - this is why we set the properties ourselves if
    // the rect is in such a state.
    if (nullRect(rect)) {
        
        ri.prev();
        r = ri.current();
        
        // r is null if we were already at the beginning
        // of the paragraph.
        if (r === null) {
            rect = {
                top: nodeRect.top,
                right: nodeRect.left,
            }
            x = nodeRect.left;
        } else {
            range.setStart(r.node, r.offset);
            range.setEnd(r.node, r.offset + r.len);
            let newRect = range.getBoundingClientRect();
            rect = {
                top: newRect.top,
                right: newRect.right,
            }
            x = newRect.right;
        }
    }
    
    // If the window width hasn't changed we don't need
    // the updated line number and caret x position.
    if (window.innerWidth === target.windowWidth) {
        
        const lineOffset = lineHeight * target.line;
        const lineTop = lineOffset + nodeRect.top;
        const diff = lineTop - rect.top;
        
        // If we're on the wrong line. This happens when
        // the caret is at the end of a line because its
        // logical index is visually on the following line.
        if (diff < -editorLineThreshold) {
            ri.prev();
            r = ri.current();
            range.setStart(r.node, r.offset);
            range.setEnd(r.node, r.offset + r.len);
            rect = range.getBoundingClientRect();
            x = rect.right;
        }
    }
    
    // Since the window width has changed it's likely
    // that the caret's x position also changed and
    // possibly the line it is on as well as the total
    // number of lines in the paragraph.
    const lineOffset = rect.top - nodeRect.top;
    const line = Math.round(lineOffset / lineHeight);
    const lineTop = nodeRect.top + (line * lineHeight);
    const total = nodeRect.height / lineHeight;
    
    return {
        caretX: x,
        caretY: lineTop,
        caretHeight: lineHeight,
        targetLine: line,
        totalLines: total,
    };
}

function placeCaretAt(x, y, h) {
    
    removeCaret();
    
    let parent;
    let scrollInner = findAncestor(".scroll_inner", focusedEditor);
    
    if (scrollInner) {
        parent = scrollInner;
    } else {
        parent = q("body");
    }
    
    // Adjust the coordinates to accomodate offset
    // and scrolled elements.
    let rect = parent.getBoundingClientRect();
    x -= rect.left;
    y -= rect.top - parent.scrollTop;
    
    let caret = document.createElement("div");
    caret.id = "caret";
    caret.style.left   = x + "px";
    caret.style.top    = y + "px";
    caret.style.height = h + "px";
    
    parent.appendChild(caret);
    syncTextarea();
}

// Updates the hidden textarea element that triggers
// the input event. We have to keep it in sync with
// the editor otherwise incorrect input events will
// trigger or none may trigger at all.
function syncTextarea() {
    
    let startOverall = 0;
    let endOverall = 0;
    
    const ordSel = orderedSelection();    
    const start = ordSel.start;
    const end = ordSel.end;
    const paras = start.paragraphs;
    
    // Note: Why would paras be undefined here?
    if (paras === undefined) {
        return;
    }
    
    for (const p of paras.slice(0, start.p)) {
        if (p.textContent === edEmptyChar) {
            startOverall += 1;
        } else {
            startOverall += p.textContent.length+1;
        }
    }
    startOverall += start.idx;
    
    if (end) {
        for (const p of paras.slice(0, end.p)) {
            if (p.textContent === edEmptyChar) {
                endOverall += 1;
            } else {
                endOverall += p.textContent.length+1;
            }
        }
        endOverall += end.idx;
        
    } else {
        endOverall = startOverall;
    }
    
    const textarea = q("textarea", focusedEditor);
    textarea.focus();
    textarea.setSelectionRange(startOverall, endOverall);
}

function removeCaret() {
    let oldCaret = q("#caret");
    if (oldCaret) {
        oldCaret.parentNode.removeChild(oldCaret);
    }
}

function editorLineHeight(node) {
    switch (node.tagName.toLowerCase()) {
        case "p":
        case "blockquote":
        case "li":
            return q("#editor_p").clientHeight;
        case "h2":
            return q("#editor_h2").clientHeight;
        case "code":
            return q("#editor_code").clientHeight;
    }
}

function lineFromPoint(node, y) {
    
    const lineHeight = editorLineHeight(node);
    const lineCount = node.clientHeight / lineHeight;
    const rect = node.getBoundingClientRect();
    const offset = y - rect.top;
    
    // Clamp line number.
    let lineAtPoint = Math.floor(offset / lineHeight);
    if (lineAtPoint < 0) {
        lineAtPoint = 0;
    }
    if (lineAtPoint >= lineCount) {
        lineAtPoint = lineCount-1;
    }
    
    const lineTop = rect.top + (lineHeight * lineAtPoint);
    const avgLineLength = Math.round((node.textContent.length-1) / lineCount);
    
    return {
        avgLen: avgLineLength,
        top: lineTop,
        left: rect.left,
        width: rect.width,
        last: lineAtPoint === lineCount-1,
        height: lineHeight,
        atPoint: lineAtPoint,
    };
}

function setCaretFromPoint(x, y) {
    
    const input = q(".input", focusedEditor);
    const paragraph = paragraphFromPoint(input, y);
    
    let line;
    let caret;
    
    if (paragraph.node.textContent === edEmptyChar) {
        
        const rect = paragraph.node.getBoundingClientRect();
        
        line = {  
            top: rect.top,
            atPoint: 0,
            last: true,
            height: rect.height,
        };
        
        caret = {   
            x: rect.left,
            idx: 0,
        };
        
    } else {
        line = lineFromPoint(paragraph.node, y);
        const xOffset = x - paragraph.node.getBoundingClientRect().left;
        const lineStartIdx = line.avgLen * line.atPoint;
        const xPos = clamp(xOffset / line.width, 0, 1);
        let approxIdx = Math.round(lineStartIdx + (xPos * line.avgLen));
        approxIdx = clamp(approxIdx, 0, paragraph.node.textContent.length-1);
        caret = caretFromPoint(paragraph.node, x, line.top, approxIdx);
    }
    
    placeCaretAt(caret.x, line.top, line.height);
    
    return {
        node: paragraph.node,
        p: paragraph.idx,
        paragraphs: paragraph.paragraphs,
        lastP: paragraph.idx === paragraph.paragraphs.length-1,
        idx: caret.idx,
        line: line.atPoint,
        lastLine: line.last,
        caretX: caret.x,
        caretY: line.top,
        caretHeight: line.height,
        windowWidth: window.innerWidth,
    };
}

function caretFromPoint(node, x, lineTop, approxIdx) {
    
    const ri = new RuneIterator(node, approxIdx);
    const range = document.createRange();
    
    let iterBackward = false;
    let closestX;
    let closestIdx;
    let caretX;
    
    // Essentially the max code points per paragraph.
    // Keep in mind this includes code blocks and astral
    // plane code points.
    let maxIterations = 4096;
    let loopCount = 0;
    
    // If approxIdx is not on the correct line we
    // iterate until we find a rune that is.
    while (true) {
        
        loopCount++;
        if (loopCount > maxIterations) {
            log(1);
            log("breaking infinite loop");
            break;
        }
        
        const r = ri.current();
        
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset + r.len);
        
        const rect = range.getBoundingClientRect();
        const diff = lineTop - rect.top;
        
        if (diff < -editorLineThreshold) {
            ri.prev();
            continue;
        }
        if (diff > editorLineThreshold) {
            ri.next();
            continue;
        }
        
        // At this point we're on the correct line so
        // we check if x is within the bounds of the
        // current idx.
        const mid = rect.left + ((rect.right - rect.left) / 2);
        if (x >= rect.left && x < mid) {
            return {
                x: rect.left,
                idx: r.overall,
            };
        }
        if (x >= mid && x <= rect.right) {
            return {
                x: rect.right,
                idx: r.overall + r.len,
            };
        }
        
        // Otherwise we specify the iteration direction.
        if (x < rect.left) {
            closestX = Math.abs(rect.left - x);
            closestIdx = r.overall;
            caretX = rect.left;
            iterBackward = true;
            ri.prev();
        } else {
            closestX = Math.abs(rect.right - x);
            closestIdx = r.overall + r.len;
            caretX = rect.right;
            ri.next();
        }
        
        break;
    }
    
    loopCount = 0;
    
    while (true) {
        
        loopCount++;
        if (loopCount > maxIterations) {
            log(2);
            log("breaking infinite loop");
            break;
        }
        
        // Iterate to the next character.
        if (iterBackward) {
            ri.prev();
        } else {
            ri.next();
        }
        
        let r = ri.current();
        
        if (r === null) {
            return {
                x: caretX,
                idx: closestIdx,
            };
        }
        
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset + r.len);
        const rect = range.getBoundingClientRect();
        const diff = Math.abs(lineTop - rect.top);
        
        // If we've moved to a different line we
        // return the closest x and idx we found.
        if (diff > editorLineThreshold) {
            return {
                x: caretX,
                idx: closestIdx,
            };
        }
        
        // If x is inside the bounding box we return.
        let mid = rect.left + ((rect.right - rect.left) / 2);
        if (x >= rect.left && x < mid) {
            return {
                x: rect.left,
                idx: r.overall,
            };
        }
        if (x >= mid && x <= rect.right) {
            return {
                x: rect.right,
                idx: r.overall + r.len,
            };
        }
        
        let delta;
        let offsetPx;
        let offsetIdx;
        
        if (x < rect.left) {
            delta = Math.abs(rect.left - x);
            offsetPx = rect.left;
            offsetIdx = 0;
        } else {
            delta = Math.abs(rect.right - x);
            offsetPx = rect.right;
            offsetIdx = r.len;
        }
        
        // If we've moved further away from the previous
        // position we return the previous position.
        if (delta >= closestX) {
            return {
                x: caretX,
                idx: closestIdx,
            };
        }
        
        // Otherwise we update the values.
        closestX = delta;
        closestIdx = r.overall + offsetIdx;
        caretX = offsetPx;
    }
}