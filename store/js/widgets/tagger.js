
function taggerFocus(e) {
    const ta = e.target;
    const tagger = findAncestor(".tagger", ta);
    tagger.classList.add("focused");
}
function taggerBlur(e) {
    const ta = e.target;
    const tagger = findAncestor(".tagger", ta);
    tagger.classList.remove("focused");
}

function taggerMouseDown(e) {

    if (findAncestor(".tag", e.target)) {
        return;
    }
    
    /*
        We call preventDefault to stop right-clicks
        resulting in caret placement. Otherwise the
        user can right-click to the zeroth index
        before our zero-width space character and be
        unable to backspace tags because that event
        will only fire if the caret isn't on index 0.
    */
    e.preventDefault();
    
    // Return if it wasn't the left mouse button.
    if (e.button !== 0) {
        return;
    }
    
    const tagger = findAncestor(".tagger", e.target);
    const input = q(".input", tagger);
    tagger.addEventListener("mousemove", taggerMouseMove);
    tagger.addEventListener("mouseup", taggerMouseUp);
    
    const idx = taggerCaretIdx(e.clientX, tagger);
    input.setSelectionRange(idx, idx);
    input.focus();
}

function taggerMouseMove(e) {
    const tagger = findAncestor(".tagger", e.target);
    const input = q(".input", tagger);
    const idx = taggerCaretIdx(e.clientX, tagger);
    input.setSelectionRange(idx, idx);
    input.focus();
}

function taggerMouseUp(e) {
    
    const tagger = findAncestor(".tagger", e.target);
    const input = q(".input", tagger);
    
    tagger.removeEventListener("mousemove", taggerMouseMove);
    tagger.removeEventListener("mouseup", taggerMouseUp);
    
    const idx = taggerCaretIdx(e.clientX, tagger);
    input.setSelectionRange(idx, idx);
    input.focus();
}

function taggerCaretIdx(x, tagger) {
    
    const input = q(".input", tagger);
    const ruler = q(".ruler", tagger);
    const range = document.createRange();
    const ri = new RuneIterator(ruler);
    
    // Compensate for distance between input and ruler.
    const iRect = input.getBoundingClientRect();
    const rRect = ruler.getBoundingClientRect();
    const offsetX = iRect.left - rRect.left;
    x -= offsetX;
    
    let r = ri.current();
    let closestX;
    let closestIdx = 0;
    let end = false;
    
    while (!end) {
        
        range.setStart(r.node, r.offset);
        range.setEnd(r.node, r.offset+r.len);
        const rect = range.getBoundingClientRect();
        
        const deltaLeft  = Math.abs(x - rect.left);
        const deltaRight = Math.abs(x - rect.right);
        const delta = deltaLeft < deltaRight ? deltaLeft : deltaRight;
        
        if (closestX === undefined) {
            closestX = delta;
            continue;
        }
        if (delta > closestX) {
            break;
        }
        
        closestX = delta;
        closestIdx = r.overall;
        
        end = ri.next() === null;
        r = ri.current();
    }
    
    if (closestIdx === 0) {
        closestIdx = 1;
    }
    
    return closestIdx;
}

/*
    The purpose of this function is to prevent
    the user from ever navigating to the zeroth
    index of the textarea. This ensures backspace
    keydowns cause the input event to fire.
*/
function taggerKeydown(e) {
    
    const input = e.target;
    
    switch (e.key) {
        
    case "PageUp":
    case "PageDown":
        e.preventDefault();
        return;
        
    case "Home":
    case "ArrowUp":
        e.preventDefault();
        const end = e.shiftKey ? input.selectionEnd : 1;
        input.setSelectionRange(1, end, "backward");
        input.focus();
        break;
        
    case "ArrowLeft":
        if (clampAfterFirstChar(input, e.shiftKey)) {
            e.preventDefault();
        }
    }
}

function clampAfterFirstChar(input, expand) {
    let clamped = false;
    let start = input.selectionStart;
    let end = input.selectionEnd;
    if (start === 1) {
        clamped = true;
        start = 1;
    }
    if (end === 1 && !expand) {
        clamped = true;
        end = 1;
    }
    input.setSelectionRange(start, end, "backward");
    return clamped;
}

function taggerInput(e) {
    
    const input = e.target;
    const tagger = findAncestor(".tagger", e.target);
    const ruler = q(".ruler", tagger);
    
    if (e.inputType === "insertLineBreak") {
        addTag(input, ruler);
        return;
    }
    
    /*
        If the textarea is empty and you type one
        character and then hit enter it won't register
        as "insertLineBreak". If you hit enter right
        away or type more than one character and *then*
        hit enter it identifies the inserted line break
        correctly. I don't know why this happens.
    */
    if (e.inputType === "insertText" && e.data === null) {
        addTag(input, ruler);
        return;
    }
    
    const backspace = e.inputType === "deleteContentBackward";
    const del = e.inputType === "deleteContentForward";
    if (backspace || del) {
        taggerDelete(input, ruler);
        return;
    }
    
    setTagInputWidth(input, ruler);
    q(".placeholder", tagger).style.display = "none";
}

function taggerDelete(input, ruler) {

    if (input.selectionStart === 0) {
        removeLastTag(input);
        return;
    }
    
    if (input.value === edEmptyChar) {
        setInputEmpty(input, ruler);
        return;
    }
    
    setTagInputWidth(input, ruler);
}

function setInputEmpty(input, ruler) {
    input.value = edEmptyChar;
    ruler.textContent = input.value;
    input.setSelectionRange(1, 1);
    const tagger = findAncestor(".tagger", input);
    q(".placeholder", tagger).style.display = "";
}

function removeTag(e) {
    const tag = findAncestor(".tag", e.target);
    tag.parentNode.removeChild(tag);
}

function removeLastTag(input) {
    const tagger = findAncestor(".tagger", input);
    const numChildren = tagger.children.length;
    setInputEmpty(input, q(".ruler", tagger));
    if (numChildren === 1) {
        return;
    }
    const lastTag = tagger.children[numChildren-2];
    tagger.removeChild(lastTag);
}

function setTagInputWidth(input, ruler) {
    ruler.textContent = input.value;
    input.style.width = ruler.clientWidth + "px";
}

function addTag(input, ruler) {
    
    const isKeyworder = findAncestor(".keyworder", input);
    
    /*
        Prevent tags being added when they
        exceed the maximum allowed.
    */
    const tagger = findAncestor(".tagger", input);
    const tags = qAll(".tag", tagger);
    const add = tagger.dataset.add ? parseInt(tagger.dataset.add) : false;
    if (add && tags.length >= add) {
        input.value = input.value.replace(/\n+/, "");
        if (isKeyworder) {
            input.dataset.tip = "Max terms reached."
        } else {
            input.dataset.tip = "Max tags reached."
        }
        tooltipEnter({target: input});
        setTimeout(() => { tooltipDismiss() }, 1000 * 3);
        return;
    }
    
    let text = input.value;
    
    // Remove zero-width space at start.
    text = text.slice(1); 
    
    // If the user hit enter in the middle
    // of the string .trim() won't be enough
    // to remove the newline, hence a regex.
    text = text.replace(/[\n.]+/, "");
    
    setInputEmpty(input, ruler);
    setTagInputWidth(input, ruler);
    
    if (text === "") {
        return;
    }
    
    const tag = document.createElement("div");
    const span = document.createElement("span");
    const remove = document.createElement("div");
    
    tag.classList.add("tag");
    span.classList.add("text");
    remove.classList.add("remove");
    remove.addEventListener("click", removeTag);
    
    span.textContent = text;
    
    tag.appendChild(span);
    tag.appendChild(remove);
    
    insertBefore(tag, findAncestor(".container", input));
    input.focus();
}