
function dropdownFocus(e) {
    
    const input = e.target;
    const dropdown = findAncestor(".dropdown", input);
    const list = q(".list", dropdown);
    dropdown.classList.remove("error");
    
    list.style.height = "";
    const items = qAll(".item", dropdown);
    for (const item of items) {
        item.classList.remove("hidden");
    }
    
    const empty = q(".empty", list);
    if (empty) {
        if (items.length === 0) {
            empty.classList.remove("hidden");
        } else {
            empty.classList.add("hidden");
        }
    }
    
    input.select();
    
    dropdownUpdateListHeight(dropdown, false);
}

function dropdownItem(e) {
    
    /*
        This prevents the blur event attached to the
        dropdown's input from firing. Otherwise the
        moment the input is blurred its handler will
        dismiss the dropdown list. That tends to feel
        too quick and doesn't give the user a chance
        to deliberate after pressing the left mouse
        button.

        We call this before checking the button to prevent
        right-clicking list items from dismissing it.
    */
    e.preventDefault();

    if (e.button !== 0) {
        return;
    }
    
    /*
        Event listeners must be named in order to be
        removable, hence we define dropdownItemUp with
        a name to avoid piling up anonymous handlers
        on the window.
    */
    window.addEventListener("mouseup", dropdownItemUp);
    const list = e.currentTarget;
    const dropdown = findAncestor(".dropdown", e.currentTarget);
    const input = q("input", dropdown);
    function dropdownItemUp(e) {
        
        window.removeEventListener("mouseup", dropdownItemUp);
        
        const upDropdown = findAncestor(".dropdown", e.target);
        const upList = findAncestor(".list", e.target);
        
        /*
            If the target does not have both a .list and
            .dropdown ancestor we do nothing. We check for
            both since it's possible a .list class could
            exist outside a .dropdown subtree.
        */
        if (!(upDropdown && upList)) {
            return;
        }
        
        // Do nothing if the list is empty.
        const empty = q(".scroll_inner > .empty:not(.hidden)", list);
        if (empty) {
            return;
        }
        
        const item = findAncestor(".item", e.target);
        
        // If we clicked on the (empty) scrollbar.
        if (!item) {
            return;
        }
        
        const icon = q(".icon", dropdown);
        const text = itemToText(item, true);
        
        if (icon) {
            const svg = q(".icon svg", item).cloneNode(true);
            icon.innerHTML = "";
            icon.appendChild(svg);
        }
        input.value = text;
        if (item.dataset.data) {
            input.dataset.data = item.dataset.data;
        }
        
        const event = new Event("update");
        input.dispatchEvent(event);
        
        /*
            If mousedown and mouseup targets share a
            common dropdown list ancestor the user has
            chosen an item, so we dismiss the list. We
            do this at the end otherwise the blur event
            handler will fire before we've finished here.
        */
        if (list === upList) {
            input.blur();
        }
    }
}

function dropdownInput(e) {
    
    const input = e.target;
    const inputValue = input.value.toLowerCase();
    const dropdown = findAncestor(".dropdown", input);
    const items = qAll(".item", dropdown);
    const icon = q(".icon", dropdown);
    
    if (icon) {
        icon.innerHTML = "";
    }
    
    let partialMatch = false;
    let exactMatchAt = -1;
    
    for (let i = 0; i < items.length; i++) {
        
        const item = items[i];
        const text = itemToText(item, false);
        
        if (item.dataset.data && inputValue === text) {
            exactMatchAt = i;
        }
        
        if (text.includes(inputValue)) {
            partialMatch = true;
            dropdown.classList.remove("error");
            item.classList.remove("hidden");
        } else {
            item.classList.add("hidden");
        }
    }
    
    if (exactMatchAt !== -1) {
        input.dataset.data = items[exactMatchAt].dataset.data;
    } else {
        input.removeAttribute("data-data");
    }
    
    if (!partialMatch && inputValue !== "") {
        dropdown.classList.add("error");
    } else {
        dropdown.classList.remove("error");
    }
    
    dropdownUpdateListHeight(dropdown, true);
    
    const event = new Event("update");
    input.dispatchEvent(event);
}

function dropdownUpdateListHeight(dropdown, anim) {
    
    let height = 0;
    let maxHeight = rem * 21;
    
    const items = qAll(".item", dropdown);
    for (const item of Array.from(items)) {
        if (item.classList.contains("hidden")) {
            continue;
        }
        height += item.clientHeight;
    }
    
    const list = q(".list", dropdown);
    const empty = q(".empty", list);
    if (items.length === 0 && empty) {
        height += empty.clientHeight;
    }
    
    if (height > maxHeight) {
        height = maxHeight;
    }
    
    list.style.height = height + "px";
    
    updateScroll(q(".scroll", dropdown), anim, true);
}

function dropdownBlur(e) {
    
    const input = e.target;
    const inputValue = input.value.toLowerCase();
    
    if (inputValue === "") {
        return;
    }
    
    const dropdown = findAncestor(".dropdown", input);
    const items = qAll(".item", dropdown);
    for (const item of Array.from(items)) {
        const text = itemToText(item, false);
        if (inputValue === text) {
            return;
        }
    }
    
    dropdown.classList.add("error");
}

function itemToText(item, preserveCase) {
    let text = [];
    for (const c of Array.from(item.children)) {
        let s = c.textContent.trim();
        if (s === "") {
            continue;
        }
        if (!preserveCase) {
            s = s.toLowerCase();
        }
        text.push(s);
    }
    return text.join(" ");
}
