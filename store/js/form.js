
function addInstance(e, kind) {
    
    const btn = findAncestor(".button", e.target);
    const adder = findAncestor(".adder", btn);
    const btnInstance = q("."+kind, btn);
    const newInstance = btnInstance.cloneNode(true);
    
    /*
        Allow inputs to be interacted with again.
    */
    const inputs = qAll("input", newInstance);
    const textareas = qAll("textarea", newInstance);
    for (const e of inputs.concat(textareas)) {
        if (findAncestor(".button", e)) {
            continue;
        }
        if (findAncestor(".disabled", e)) {
            continue;
        }
        e.removeAttribute("disabled");
    }
    
    /*
        Since groups have their own "add" icon we need
        to replace it with the ordinary icon. We do this
        by selecting the first group and cloning its icon.
        This is safe to do because there's always at least
        one instance of a group.
    */
    if (kind === "group") {
        const icon = q(".legend .icon", newInstance);
        const firstGroupIcon = q(".group .legend .icon", adder);
        replaceNode(firstGroupIcon.cloneNode(true), icon);
    }

    insertBefore(newInstance, btn);
    updateInstances(adder, kind);
    init(newInstance);
}

function removeInstance(e, kind) {
    e.stopPropagation();
    const instance = findAncestor("."+kind, e.target);
    const adder = findAncestor(".adder", instance);
    removeNode(instance);
    updateInstances(adder, kind);
}

function updateInstances(adder, kind) {
    
    const max = parseInt(adder.dataset.add);
    const n = adder.children.length-1;
    
    setInstanceRemoves(adder);
    
    const btn = adder.children[n];
    if (n === max) {
        btn.classList.add("hidden");
    } else {
        btn.classList.remove("hidden");
    }
    
    const scroll = findAncestor(".scroll", adder);
    if (scroll) {
        updateScroll(scroll, true, true);
    }
    
    if (n === max) {
        return;
    }
    
    for (let i = 0; i < adder.children.length; i++) {
        
        let child = adder.children[i];
        
        if (child.classList.contains("button")) {
            child = q("."+kind, child);
        }
        
        let num = i+1;
        let radioIdxPos;
        switch (kind) {
        case "group":
            radioIdxPos = -3;
            break;
        case "subgroup":
            num = intToRoman(num);
            radioIdxPos = -2;
            break;
        case "instance":
            num = intToAlpha(num);
            radioIdxPos = -1;
            break;
        }

        const radioBtns = qAll(`input[type="radio"]`, child);
        for (const rb of radioBtns) {
            const parts = rb.getAttribute("name").split("_");
            const n = parts.length;
            parts[parts.length + radioIdxPos] = ""+i;
            rb.setAttribute("name", parts.join("_"));
        }
        
        q(".num", child).textContent = num;
    }
}

function setInstanceRemoves(adder) {
    
    const n = adder.children.length-1;
    const children = Array.from(adder.children);
    const instances = children.slice(0, -1);
    
    for (const instance of instances) {
    
        const last = instance.children.length-1;
        const remove = instance.children[last];
    
        if (n === 1) {
            remove.classList.add("hidden");
        } else {
            remove.classList.remove("hidden");
        }
    }
}

function intToAlpha(n) {
    const alpha = "abcdefghijklmnopqrstuvwxyz";
    return alpha[n-1];
}

function intToRoman(n) {

    const multiples = [
        [1000, "M"], [900, "CM"],
        [500, "D"], [400, "CD"],
        [100, "C"], [90, "XC"],
        [50, "L"], [40, "XL"],
        [10, "X"], [9, "IX"],
        [5, "V"], [4, "IV"],
        [1, "I"],
    ];

    let s = "";
    for (const m of multiples) {
        s += m[1].repeat(Math.floor(n/m[0]));
        n %= m[0];
    }

    return s;
}

function setGroupVisibility(e, groupNamePrefix) {
    
    const input = e.target;
    const group = findAncestor(".group", input);
    const name = input.getAttribute("name");
    const inputs = qAll(`[name="${name}"]`, group);
    const dropdown = findAncestor(".dropdown", input);
    const items = qAll(".item", dropdown);
    const form = findAncestor("form", group);
    const dataKinds = {};
    
    // Populate dataKinds with active groups.
    for (const input of inputs) {
        if (input.dataset.data) {
            dataKinds[input.dataset.data] = true;
        }
    }
    
    // Populate dataKinds with inactive groups.
    for (const item of items) {
        
        if (!item.dataset.data) {
            continue;
        }
        
        if (!dataKinds[item.dataset.data]) {
            dataKinds[item.dataset.data] = false;
        }
    }
    
    for (const data in dataKinds) {
        
        let name = data;
        if (groupNamePrefix) {
            name = groupNamePrefix + "_" + name;
        }
        
        const visible = dataKinds[data];
        const group = q(`.group[name="${name}"]`, form);
        if (visible) {
            group.classList.remove("hidden");
        } else {
            group.classList.add("hidden");
        }
    }
    
    updateScroll(findAncestor(".scroll", form), true);
}

/*
    This is intended to populate a field placeholder that is related
    to the target dropdown, hence destName refers to the name attribute
    of a sibling of the target dropdown or to one of the descendents
    of said dropdown's parent. It will not search the entire form or DOM.
*/
function requestFieldFromDropdown(e, destName, prefix) {

    const input = e.currentTarget;
    const dropdown = findAncestor(".dropdown", input);
    const item = itemFromDropdownValue(dropdown);
    const col = findAncestor(".col", dropdown);
    
    if (!item) {
        return;
    }
    
    const fieldParent = findAncestor(".field", dropdown).parentNode;
    const destinations = formFields(destName, fieldParent, false);
    for (const dest of destinations) {
        enableElem(dest);
        dest.classList.add("empty");
        dest.innerHTML = loading;
    }
    
    let name = item.dataset.data;
    if (prefix) {
        name = prefix + "." + name;
    }
    
    const view = context.subView ? context.subView : context.view;
    
    get(`/${view}/${col.id}/field/${name}/partial`, (err, res) => {
        
        if (err) {
            throw err;
            return;
        }
        
        for (const dest of destinations) {
            dest.innerHTML = res;
            dest.classList.remove("empty");
            init(dest);
        }
    });
}

function enableElem(elem) {
    while (true) {
        elem = findAncestor(".disabled", elem);
        
        /*
            Remove disabled *attritube* (not class)
            from children of element that has the
            "disabled" class.
        */
        const disabled = qAll("[disabled]", elem);
        for (const d of disabled) {
            d.removeAttribute("disabled");
        }
        
        if (!elem) {
            break;
        }
        elem.classList.remove("disabled");
    }
}

function itemFromDropdownValue(dropdown, cloneItem) {

    const input = q("input", dropdown);
    const items = qAll(".item", dropdown);

    for (let item of items) {
        
        const text = q(".text", item);
        if (text.textContent !== input.value) {
            continue;
        }
        
        if (cloneItem) {
            item = item.cloneNode(true);
        }
        
        return item;
    }
}

function disallowSiblingDuplicates(e, fieldName) {

    const dropdown = findAncestor(".dropdown", e.target);
    const form = findAncestor("form", dropdown);
    const fields = formFields(fieldName, form, true);
    const seenValue = {};

    for (const tf of fields) {
        if (tf === e.target) {
            continue;
        }
        seenValue[tf.value] = true;
    }

    const items = qAll(".item", dropdown);
    for (let i = items.length-1; i >= 0; i--) {
        const item = items[i];
        const text = q(".text", item);
        if (seenValue[text.textContent]) {
            removeNode(item);
        }
    }
}

function populateDropdownFromTextInputs(e, fieldName) {

    const dropdown = findAncestor(".dropdown", e.target);
    const form = findAncestor("form", dropdown);
    const fields = formFields(fieldName, form, true);
    const listInner = getListInner(dropdown);
    const seenValue = {};
    
    for (const c of Array.from(listInner.children)) {
        if (c.classList.contains("empty")) {
            continue;
        }
        removeNode(c);
    }
    
    for (let i = 0; i < fields.length; i++) {
        
        let val = fields[i].value;
        if (val === "") {
            continue;
        }

        // Disallow duplicates.
        if (seenValue[val]) {
            continue;
        }
        seenValue[val] = true;
        
        const srcDropdown = findAncestor(".dropdown", fields[i]);
        if (!srcDropdown) {
            const item = itemFromText(val, fields[i].dataset.data);
            listInner.appendChild(item);
            continue;
        }
        
        /*
            If the text input is part of a dropdown we check to
            see if the dropdown actually contains the text input's
            value. If not, we skip it.
        */
        const items = qAll(".item", srcDropdown);
        let clonedItem;
        for (const item of items) {
            const text = q(".text", item);
            if (text.textContent === val) {
                clonedItem = item.cloneNode(true);
                break;
            }
        }
        if (!clonedItem) {
            continue;
        }
        listInner.appendChild(clonedItem);
    }
}

function formFields(name, elem, noButton) {
    
    name = name.split(".");
    if (name.length === 0) {
        throw "expected 'name' argument to have at least 1 part";
    }
    
    const form = findAncestor("form", elem);
    const lastName = name[name.length-1];
    const fields = qAll(`[name="${lastName}"]`, elem);
    const newFields = [];
    const has = (e, cls) => e.classList.contains(cls);
    
    for (const f of fields) {
        
        if (noButton && findAncestor(".button", f)) {
            continue;
        }
        
        const path = [];
        let current = f;
        
        while (true) {
            
            if (current === form) {
                break;
            }
            
            if (adderOfSameName(path, current, lastName)) {
                current = current.parentNode;
                continue;   
            }
            
            if (current.hasAttribute("name") && !has(current, "field")) {
                path.unshift(current.getAttribute("name"));
            }
            
            if (pathsMatch(name, path)) {
                newFields.push(f);
                break;
            }
            
            if (!pathsCouldMatch(name, path)) {
                break;
            }
            
            current = current.parentNode;
        }
    }
    
    return newFields;
}

function adderOfSameName(path, elem, name) {
    const has = (e, cls) => e.classList.contains(cls);
    if (!has(elem, "adder")) {
        return false;
    }
    if (has(elem, "grp") || has(elem, "sg")) {
        return false;
    }
    if (path.length > 1) {
        return false;
    }
    if (!elem.hasAttribute("name")) {
        return false;
    }
    if (elem.getAttribute("name") !== name) {
        return false;
    }
    return true;
}

function pathsMatch(p1, p2) {
    if (p1.length !== p2.length) {
        return false;
    }
    for (let i = 0; i < p1.length; i++) {
        if (p1[i] !== p2[i]) {
            return false;
        }
    }
    return true;
}

// p1 is the target path, p2 is the incomplete path
function pathsCouldMatch(p1, p2) {
    
    if (p2.length > p1.length) {
        return false;
    }
    
    // We only compare the corresponding elements.
    const diff = p1.length - p2.length;
    p1 = p1.slice(diff);
    for (let i = 0; i < p2.length; i++) {
        if (p2[i] !== p1[i]) {
            return false;
        }
    }
    
    return true;
}

/*
    Account for uninitialised scroll in add button dropdown.
    We don't query for an item and grab its parent because
    there might not be any items in the list.
*/
function getListInner(dropdown) {
    let listInner = q(".list .scroll_inner", dropdown);
    if (!listInner) {
        listInner = q(".list .scroll", dropdown);
    }
    return listInner;
}

function itemFromText(s, data) {
    
    const item = document.createElement("div");
    const text = document.createElement("div");
    
    if (data) {
        item.dataset.data = data;
    }
    
    item.appendChild(text);
    item.classList.add("item");
    text.classList.add("text");
    text.textContent = s;
    
    return item;
}