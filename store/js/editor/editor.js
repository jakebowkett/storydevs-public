/*
    TODO:
        Attach editor paragraphs to "ed" rather than "ed.anchor" or "ed.focus"
        You can still bring up the link editor for headings and quotes
        new line bounding rect is always on the preceding line which is visually confusing
*/

function setupCanvas() {
    const canvas = q("canvas", focusedEditor);
    const ctx = canvas.getContext("2d");
    ctx.canvas.width  = cssPx(q("#search .col_inner"), "max-width");
    ctx.canvas.height = window.screen.height;
    ctx.lineWidth     = "1";
    ctx.fillStyle = "#008a80";
}

/*
    This is to handle tabbing through inputs. It will
    return early if the textarea being focused belongs
    to an already focused editor.
    
    Various editor operations will call the textarea's
    focus method, causing this event handler to be
    executed. It should only fully execute in the case
    of the focus being the result of tabbing.
    
    Tabbing away is handled in the editorKeydown handler.
*/
function edTextareaFocus(e) {
    
    const currentEditor = findAncestor(".ed", e.target);
    if (focusedEditor === currentEditor) {
        return;
    }
    focusedEditor = currentEditor;
    
    const input = q(".input", currentEditor);
    input.classList.add("focused");
    
    updateCanvasPos();
    setupCanvas();
    editorAddEvents();
    
    /*
        Set the top left boundary of the editor as
        the point so that the caret gets placed at
        index 0 of paragraph 0.
    */
    const rect = currentEditor.getBoundingClientRect();
    setCollapsedFromPoint(rect.left, rect.top);
    
    ed.addFormat = [];
    ed.removeFormat = [];
    updateFormatMenu(ed.anchor.node, ed.anchor.idx);
    viewLink();

    ensureCaretInView(rem * 8.5);
}

// This is called by scrollEvent in scrolls.js
function updateEditorToolsPos(elem) {
    
    const editors = qAll(".ed", elem);
    
    for (let editor of editors) {
        
        const tools = q(".tools", editor);
        const input = q(".input", editor);
        const canvas = q("canvas", editor);
        const eRect = editor.getBoundingClientRect();
        const iRect = input.getBoundingClientRect();
        const tRect = tools.getBoundingClientRect();
        const range = iRect.height - tRect.height;
        const current = cssPx(tools, "margin-top");
        
        if (eRect.bottom > iRect.bottom) {
            tools.style.marginTop = range + "px";
        }
        else if (tRect.top < rem*6) {
            let offset = current + Math.abs(tRect.top - rem*6);
            if (offset > range) {
                offset = range;
            }
            tools.style.marginTop = offset + "px";
        }
        else {
            let offset = current - Math.abs(tRect.top - rem*6);
            if (offset < 0) {
                offset = 0;
            }
            tools.style.marginTop = offset + "px";
        }  
    }
}

// This should always be called prior to drawSelection()
function updateCanvasPos() {
    
    if (!focusedEditor) {
        return;
    }
    
    const input = q(".input", focusedEditor);
    const canvas = q("canvas", focusedEditor);
    const rect = canvas.getBoundingClientRect();
    
    if (rect.top !== 0) {
        let current = cssPx(canvas, "margin-top");
        canvas.style.marginTop = current + -rect.top + "px";
    }
}

var lastTripleClick = Date.now();

function editorFocus(e) {
    
    // Return if it wasn't the left mouse button.
    if (e.button !== 0) {
        return;
    }
    
    e.preventDefault();
    
    // When an input such as a textfield or a textarea
    // is selected it stays focused due to the call to
    // preventDefault above, therefore it's necessary
    // to blur the active element.
    document.activeElement.blur();
    
    if (findAncestor(".link_input", e.target)) {
        return;
    }
    dismissLinkEditor();
    
    const currentEditor = findAncestor(".ed", e.target);
    const prevEditor = focusedEditor;
    focusedEditor = currentEditor;
    if (prevEditor !== currentEditor && prevEditor !== null) {
        editorUnfocus();
    }
    focusedEditor = currentEditor;
    const input = q(".input", focusedEditor);
    const touch = q(".touch", focusedEditor);
    input.classList.add("focused");
    touch.focus();
    
    if (e.detail === 2) {
        let now = Date.now();
        if (now - lastTripleClick > editorMultiClickThreshold) {
            selectWord();
            return;
        }
    }
    if (e.detail === 3) {
        let now = Date.now();
        if (now - lastTripleClick > editorMultiClickThreshold) {
            selectParagraph();
            lastTripleClick = now;
            return;
        }
    }
        
    updateCanvasPos();
    setupCanvas();
    
    window.addEventListener("focus", editorWindowFocus);
    window.addEventListener("blur", editorWindowBlur);
    
    window.addEventListener("mousemove", editorMouseMove);
    window.addEventListener("mouseup", editorMouseUp);
    window.addEventListener("mousedown", editorBlur);
    window.addEventListener("keydown", editorKeydown);
    window.addEventListener("input", editorInput);
    
    /*
        A storydevsCopy event is attached to the window
        on page load. If focusedEditor is not null then
        storydevsCopy will pass the event to editorCopy.
        Otherwise it will perform a normal copy but remove
        any shy hyphens from the text in case it's pasted
        back into this editor and the invisible hyphens
        fuck it up.
    */
    window.addEventListener("cut", editorCut);
    window.addEventListener("paste", editorPaste);
    
    setCollapsedFromPoint(e.clientX, e.clientY);
    
    ed.addFormat = [];
    ed.removeFormat = [];
    updateFormatMenu(ed.anchor.node, ed.anchor.idx);
    
    viewLink();
    
    syncTextarea();
}

function setCollapsedFromPoint(x, y) {
    let anchor = setCaretFromPoint(x, y);
    ed.anchor.node = anchor.node;
    ed.anchor.pEmpty = anchor.node.textContent === edEmptyChar;
    ed.anchor.p = anchor.p;
    ed.anchor.paragraphs = anchor.paragraphs;
    ed.anchor.lastP = anchor.lastP;
    ed.anchor.line = anchor.line;
    ed.anchor.idx = anchor.idx;
    ed.anchor.lastLine = anchor.lastLine;
    ed.anchor.xMem = anchor.caretX;
    ed.anchor.caretX = anchor.caretX;
    ed.anchor.caretY = anchor.caretY;
    ed.anchor.caretHeight = anchor.caretHeight;
    ed.anchor.windowWidth = anchor.windowWidth;
    
    // Selections are always collapsed on a new mousedown.
    Object.assign(ed.focus, ed.anchor);
}

function editorWindowBlur(e) {
    const caret = q("#caret");
    if (!caret) {
        return;
    }
    caret.style.display = "none";
}
function editorWindowFocus(e) {
    const caret = q("#caret");
    if (!caret) {
        return;
    }
    caret.style.display = "";
}

function editorBlur(e) {
    
    // Return if we've clicked within an editor
    // the focus handler above will handle it.
    const editor = findAncestor(".ed", e.target);
    const linkEditor = findAncestor("#link_editor", e.target);
    if (editor || linkEditor) {
        return;
    }
    
    // Otherwise, if no editor was clicked...
    editorUnfocus();
}

function editorUnfocus() {
    
    removeCaret();
    clearEditorCanvas();
    clearFormatMenu();
    dismissLinkEditor();
    // focusedEditor.classList.remove("focused");
    q(".input", focusedEditor).classList.remove("focused");
    q(".touch", focusedEditor).blur();
    
    editorRemoveEvents();
    
    ed.anchor = {};
    ed.focus = {};
    ed.addFormat = [];
    ed.removeFormat = [];
    
    focusedEditor = null;
}

function editorAddEvents() {
    
    // We omit adding the mousemove and mouseup
    // events as they only make sense being attached
    // by a mousedown handler.
    window.addEventListener("focus", editorWindowFocus);
    window.addEventListener("blur", editorWindowBlur);
    
    window.addEventListener("mousedown", editorBlur);
    window.addEventListener("keydown", editorKeydown);
    window.addEventListener("input", editorInput);
    
    window.addEventListener("cut", editorCut);
    window.addEventListener("paste", editorPaste);
}
    
function editorRemoveEvents() {
    
    window.removeEventListener("focus", editorWindowFocus);
    window.removeEventListener("blur", editorWindowBlur);
    
    window.removeEventListener("mousemove", editorMouseMove);
    window.removeEventListener("mouseup", editorMouseUp);
    window.removeEventListener("mousedown", editorBlur);
    window.removeEventListener("keydown", editorKeydown);
    window.removeEventListener("input", editorInput);
    
    window.removeEventListener("cut", editorCut);
    window.removeEventListener("paste", editorPaste);
}

function editorMouseMove(e) {
    
    let focus = setCaretFromPoint(e.clientX, e.clientY);
    
    ed.focus.node = focus.node;
    ed.focus.pEmpty = focus.node.textContent === edEmptyChar;
    ed.focus.p = focus.p;
    ed.focus.paragraphs = focus.paragraphs;
    ed.focus.lastP = focus.lastP;
    ed.focus.line = focus.line;
    ed.focus.idx = focus.idx;
    ed.focus.lastLine = focus.lastLine;
    ed.focus.xMem = focus.caretX;
    ed.focus.caretX = focus.caretX;
    ed.focus.caretY = focus.caretY;
    ed.focus.caretHeight = focus.caretHeight;
    ed.focus.windowWidth = focus.windowWidth;
    
    drawSelection();
    
    if (selectionCollapsed()) {
        viewLink();
    } else {
        dismissLinkEditor();
    }
    
    syncTextarea();
}

function editorMouseUp(e) {
    
    if (e.button !== 0) {
        return;
    }
    
    window.removeEventListener("mousemove", editorMouseMove);
    window.removeEventListener("mouseup", editorMouseUp);
    
    let focus = setCaretFromPoint(e.clientX, e.clientY);
    ed.focus.node = focus.node;
    ed.focus.pEmpty = focus.node.textContent === edEmptyChar;
    ed.focus.p = focus.p;
    ed.focus.paragraphs = focus.paragraphs;
    ed.focus.lastP = focus.lastP;
    ed.focus.line = focus.line;
    ed.focus.idx = focus.idx;
    ed.focus.lastLine = focus.lastLine;
    ed.focus.xMem = focus.caretX;
    ed.focus.caretX = focus.caretX;
    ed.focus.caretY = focus.caretY;
    ed.focus.caretHeight = focus.caretHeight;
    ed.focus.windowWidth = focus.windowWidth;
    
    drawSelection();
    
    if (selectionCollapsed()) {
        viewLink();
    } else {
        dismissLinkEditor();
    }
    
    const start = orderedSelection().start;
    updateFormatMenu(start.node, start.idx);
    
    syncTextarea();
}

