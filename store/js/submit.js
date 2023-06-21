
function updateModalHeight(form) {
    const hiddenModal = q("#modal_hidden");
    hiddenModal.innerHTML = "";
    hiddenModal.appendChild(form.cloneNode(true));
    updateScroll(findAncestor(".scroll", form), false, true);
}

/*
TODO: #acc-email no longer exists -- we need to specify
the ID in the /store/data/mode/ file that generates the
appropriate UI.

Actually!!! Maybe a special class like ".acc-email" that
allows us to update any instance of the email being displayed?
*/
function updateEmail() {
    get("/data/account/email", function(err, res) {
        if (err) {
            log(err);
            return;
        }
        q("#acc-email").textContent = res;
    });
}

function submitModal(e) {
    
    e.preventDefault();

    const form = findAncestor("form", e.target);
    const footer = q(".footer", form);
    const dest = findAncestor(".btn", e.target).dataset.dest;
    const login = dest === "login";
    const pf = parseForm(form, false);
    
    let callback = e.target.dataset.callback;
    if (callback) {
        callback = window[callback];
    }
    
    /*
        Update modal height in case errors have
        been displayed. Also clears height from
        any errors that have been removed.
    */
    updateModalHeight(form);
    
    // Return here if parseForm determines we shouldn't submit.
    if (!pf) {
        return;
    }
    
    let json = pf.get("json");
    json = JSON.parse(json);
    
    footer.appendChild(strToElem(loading));
    
    post("/" + dest, json, (err, res) => {
        
        removeNode(q(".loading", footer));
        
        // Server error.
        if (err) {
            log(err);
            return;
        }
        
        // User error.
        if (res && res.feedback) {
            addFeedback(res.feedback, form);
            updateModalHeight(form);
            scrollToFirstError(form);
            return;
        }
        
        if (callback) {
            callback(res);
        }
        
        if (login) {
            
            const tmp = document.createElement("div");
            const locSect = q("#sidebar .section.location");
            
            tmp.innerHTML = res.ps;
            insertAfter(tmp.children[0], q("h4", locSect));
            init(q("#persona_switcher"));
            
            if (res.link) {
                tmp.innerHTML = res.link;
                locSect.appendChild(tmp.children[0]);
                init(locSect);
            }
            
            if (res.meta) {
                for (const meta of res.meta) {
                    context.mode[meta.name] = {
                        title: meta.title,
                        resourceName: meta.resourceName,
                        resourcePlural: meta.resourcePlural,
                        resourceColumn: meta.resourceColumn,
                        logoutRemove: meta.logoutRemove,
                    };
                }
            }
            
            context.loggedIn = true;
            q("#auth").innerHTML = ".logged_out { display: none !important; }";
            dismissModal();
            
            return;
        }
        
        if (!res || !res.modal) {
            dismissModal();
            return;
        }
        
        const modal = q("#modal");
        emptyModal(modal);
        populateModal(modal, res.modal);
        const inner = q(".inner", modal);
        
        const firstInput = q("input", inner);
        if (firstInput) {
            firstInput.focus();
        }
        
        init(inner);
    });
}

function updateResource(e) {
    submitResource(e, submitRoute(true), true);
}

function createResource(e) {
    submitResource(e, submitRoute(false));
}

function submitRoute(update) {
    let route;
    if (context.subView) {
        route = "/" + context.view + "/" + context.subView;
    } else {
        route = "/" + context.view;
    }
    if (update) {
        route += "/" + context.editing;
    }
    return route;
}

function submitResource(e, path, update) {
    
    e.preventDefault();
    
    const form = findAncestor("form", e.target);
    const footer = q("#editor_footer", form);
    
    const pf = parseForm(form);
    if (!pf) {
        return;
    }
    
    footer.appendChild(strToElem(loading));
    
    put(path, pf, (err, res) => {
        
        removeNode(q(".loading", footer));
        
        const submitKind = update ? "update" : "create";
        const c = context;
        let resourceName;
        if (c.subView) {
            if (c.subView === "settings") {
                resourceName = c.resource;
            } else {
                resourceName = c.mode[c.subView].resourceName;
            }
        } else {
            resourceName = c.mode[c.view].resourceName;
        }
        resourceName = resourceName.toLowerCase();

        if (err) {
            log(err);
            showNotification(
                "Error",
                `Unable to ${submitKind} your ${resourceName} at this time.`,
                "error"
            );
            return;
        }

        // User error.
        if (res && res.feedback) {
            addFeedback(res.feedback, form);
            scrollToFirstError(form);
            return;
        }

        showNotification(
            "Success!",
            `Your ${resourceName} has been ${submitKind}d.`,
            "success"
        );

        if (context.subView === "settings") {
            return;
        }
        
        let slug = res.slug;
        let resource = res.resource;
        let reply = res.reply;
        
        if (c.subView) {
            let route = `/${c.view}/${c.subView}/${slug}`;
            history.pushState({}, "", reply ? `${route}#post-${reply}` : route);
            setLayout("detail", c.view, c.subView);
        } else {
            let route = `/${c.view}/${slug}`;
            history.pushState({}, "", reply ? `${route}#post-${reply}` : route);
            setLayout("search-detail", c.view);
        }
        context.resource = slug;
        setEmpty(context.view, ["editor"]);
        
        q("#detail .col_inner").innerHTML = resource;
        q("#detail .scroll_inner").scrollTop = 0;
        
        const detail = q("#detail");
        init(detail);
        if (reply) {

            if (detail.dataset.moving) {
                detail.addEventListener("canload", load);
            } else {
                load();
            }

            function load(e) {
                scrollTo(q(`#post-${reply}`), {padding: scrollToPad});
            }
        }
    });
}

function parseForm(form, search) {
    
    clearErrors(form);
    const data = {};
    const fb = {};
    const has = (e, cls) => e.classList.contains(cls);
    const fd = new FormData();
    const chain = [];
    let canSubmit = true;
    
    function recurse(node, target, multi) {
        
        let idx = 0;
        
        for (const c of Array.from(node.children)) {
            
            if (has(c, "button")) {
                continue;
            }
            
            if (has(c, "adder")) {
                
                const k = search ? c.dataset.id : elemName(c);
                const t = [];
                target[k] = t;
                chain.push(k);
                recurse(c, t, true);
                chain.pop();
                
                if (t.length === 0) {
                    delete target[k];
                }
                
                continue;
            }
            
            if (has(c, "group") || has(c, "subgroup")) {
                
                if (!multi) {
                    if (c.dataset.submitSingle) {
                        const t = {};
                        const k = search ? c.dataset.id : elemName(c);
                        target[k] = t;
                        recurse(c, t, false);
                        continue
                    }
                    recurse(c, target, false);
                    continue;
                }
                
                const t = {};
                target.push(t);
                
                chain.push(idx);
                idx++;
                
                recurse(c, t, false);
                
                if (keys(t).length === 0) {
                    target.pop();
                }
                chain.pop();
                
                continue;
            }
               
            let k, v, ok, seen;
            
            /*
                Parse[FieldType] returns that field's value. If there was no
                value then val will be null. If val is null and the field was
                not optional, fb will be populated with an error message.
            */
            if (has(c, "password"))   { [k, v, ok] = parsePassword(c, search);  seen = true; }
            if (has(c, "textfield"))  { [k, v, ok] = parseTextfield(c, search); seen = true; }
            if (has(c, "textarea"))   { [k, v, ok] = parseTextarea(c, search);  seen = true; }
            if (has(c, "checkbox"))   { [k, v, ok] = parseCheckbox(c, search);  seen = true; }
            if (has(c, "radio"))      { [k, v, ok] = parseRadio(c, search);     seen = true; }
            if (has(c, "bool"))       { [k, v, ok] = parseBool(c, search);      seen = true; }
            if (has(c, "dropdown"))   { [k, v, ok] = parseDropdown(c, search);  seen = true; }
            if (has(c, "calendar"))   { [k, v, ok] = parseCalendar(c, search);  seen = true; }
            if (has(c, "wdgt-time"))  { [k, v, ok] = parseTime(c, search);      seen = true; }
            if (has(c, "ed"))         { [k, v, ok] = parseEditor(c, search);    seen = true; }
            if (has(c, "tagger"))     { [k, v, ok] = parseTagger(c, search);    seen = true; }
            if (has(c, "range-wdgt")) { [k, v, ok] = parseRange(c, search);     seen = true; }
            if (has(c, "image")) {
                [k, v, ok] = parseImage(c, search);
                if (!ok) {
                    canSubmit = false;
                }
                if (typeof v === "string") {
                    addToTarget(target, k, v);
                    continue;
                }
                addToTarget(target, k, null);
                fd.append(chainString(chain, k), v);
                continue;
            }
            
            if (!seen) {
                recurse(c, target, multi);
                continue;
            }
            
            if (!ok) {
                canSubmit = false;
            }
            
            if (k === null) {
                continue;
            }
            
            addToTarget(target, k, v);
        }
    }
    
    recurse(form, data, false);
    
    if (invalidAdders(form, search)) {
        canSubmit = false;
    }

    if (!canSubmit) {
        scrollToFirstError(form);
        return false;
    }
    
    fd.set("json", JSON.stringify(data));
    return fd;
}

function scrollToFirstError(form) {
    const firstErr = q(".errors > div", form);
    const field = findAncestor(".field", firstErr);
    updateScroll(findAncestor(".scroll", form), false, true);
    scrollTo(field);
}

function addToTarget(t, k, v) {
    if (Array.isArray(t)) {
        if (v !== null) {
            t.push(v);
        }
    } else {
        t[k] = v;
    }
}

function chainString(c, k) {
    let s = c.join(".");
    if (s !== "") {
        s += "."
    }
    return s + k;
}

function textInvalidLength(elem, v, search) {
    
    if (search) {
        return false;
    }
    
    const n = len(v);
    const f = findAncestor(".field", elem);
    const min = f.dataset.min ? parseInt(f.dataset.min) : false;
    const max = f.dataset.max ? parseInt(f.dataset.max) : false;
    
    let invalid = false;
    
    if (min && n < min) {
        addError(elem, `Field requires a minimum of ${min} characters.`);
        invalid = true;
    }
    if (max && n > max) {
        addError(elem, `Field cannot exceed ${max} characters.`);
        invalid = true;
    }
    
    return invalid;
}

function ctrlChar(elem, s, search) {
    if (search) {
        return false;
    }
    if (/[\x00-\x1F\x7F-\x9F]/g.test(s)) {
        addError(elem, "Field cannot contain control characters.");
        return true;
    }
    return false
}

// Allows newlines.
function ctrlCharSansNewLine(elem, s, search) {
    if (search) {
        return false;
    }
    if (/[\x00-\x09\x0B-\x1F\x7F-\x9F]/g.test(s)) {
        addError(elem, "Field cannot contain control characters.");
        return true;
    }
    return false
}

function isWord(s) {
    const valid = /^\w+$/.test(s);
    return [valid, "Field may only contain letters, numbers, or underscores."];
}

function isEmail(s) {
    const valid = /^.+@.+\..+$/.test(s);
    return [valid, "Invalid email address."];
}

function isDiscord(s) {
    const valid = /^.+#\d{4}$/.test(s);
    return [valid, "Invalid Discord handle."];
}

function isDomain(s) {
    const valid = /^.*[a-zA-Z0-9-]+\.[a-zA-Z0-9]+.*$/.test(s);
    return [valid, "Invalid url."];
}

function invalidInput(elem, input) {
    const funcs = input.dataset.validate;
    if (!funcs) {
        return false;
    }
    let invalid = false;
    for (const func of funcs.split(", ")) {
        const [valid, msg] = window[func](input.value);
        if (!valid) {
            addError(elem, msg);
            invalid = true;
        }
    }
    return invalid;
}

function parsePassword(pw, search) {
    const input = q("input", pw);
    const opt = isOptional(input, search);
    const k = elemName(input);
    const v = input.value;
    if (!opt && !v) {
        requiredError(pw, search);
        return [null, null, false];
    }
    let invalid = false;
    if (ctrlChar(pw, v, search)) {
        invalid = true;
    }
    if (textInvalidLength(pw, v, search)) {
        invalid = true;
    }
    if (invalidInput(pw, input)) {
        invalid = true;
    }
    if (invalid) {
        return [null, null, false];
    }
    return [k, v, true];
}

function parseRange(range, search) {
    const k = search ? range.dataset.id : elemName(range);
    const [start, end] = rangeMarkerPositions(range);
    if (search) {
        return [k, [start, end-1], true];
    }
    const track = q(".track", range);
    const startText = track.children[start].getAttribute("name");
    const endText = track.children[end-1].getAttribute("name");
    const v = startText === endText ? startText : startText + "-" + endText;
    return [k, v, true];
}

function parseTextfield(tf, search) {
    const input = q("input", tf);
    const opt = isOptional(input, search);
    const k = elemName(input);
    const v = input.value;
    const err = [null, null, false];
    if (!opt && !v) {
        requiredError(tf, search);
        return err;
    }
    if (!v) {
        return [k, null, true];
    }
    if (ctrlChar(tf, v, search)) {
        return err;
    }
    if (textInvalidLength(tf, v, search)) {
        return err;
    }
    if (invalidInput(tf, input)) {
        return err;
    }
    return [k, v, true];
}

function parseTextarea(ta, search) {
    const input = q("textarea", ta);
    const opt = isOptional(input, search);
    const k = elemName(input);
    const v = input.value;
    if (!opt && !v) {
        requiredError(ta, search);
        return [null, null, false];
    }
    if (!v) {
        return [k, null, true];
    }
    let invalid = false;
    if (ctrlCharSansNewLine(ta, v, search)) {
        invalid = true;
    }
    if (textInvalidLength(ta, v, search)) {
        invalid = true;
    }
    if (invalidInput(ta, input)) {
        invalid = true;
    }
    if (invalid) {
        return [null, null, false];
    }
    return [k, v, true];
}

function parseCheckbox(container, search) {
    
    const opt = isOptional(container, search);
    const cc = qAll("input", container);
    const k = search ? cc[0].dataset.id : elemName(cc[0]);
    const v = [];
    
    for (let i = 0; i < cc.length; i++) {
        if (cc[i].checked) {
            if (search) {
                v.push(i);
            } else {
                v.push(cc[i].value);
            }
        }
    }
    
    if (v.length === 0) {
        if (opt) {
            return [null, null, true];
        } else {
            requiredError(container, search);
            return [null, null, false];
        }
    }
    
    return [k, v, true];
}

function parseRadio(container, search) {
    
    const opt = isOptional(container, search);
    const cc = qAll("input", container);
    const k = search ? cc[0].dataset.id : elemName(cc[0]);
    
    for (let i = 0; i < cc.length; i++) {
        if (cc[i].checked) {
            if (search) {
                return [k, i, true];
            } else {
                return [k, cc[i].value, true];
            }
        }
    }
    
    if (!opt) {
        requiredError(container, search);
    }

    return [null, null, true];
}

function parseBool(container, search) {
    
    const opt = isOptional(container, search);
    const cc = qAll("input", container);
    const k = search ? cc[0].dataset.id : elemName(cc[0]);
    
    let enabled = false;
    let seenChecked = false;
    let seenIdx;

    for (let i = 0; i < cc.length; i++) {
        const isTrue = cc[i].dataset.bool === "true";
        if (cc[i].checked) {
            seenIdx = i;
            seenChecked = true;
        } else {
            continue;
        }
        if (isTrue) {
            enabled = true;
        }
    }

    if (!opt && !seenChecked) {
        requiredError(container, search);
        return [null, null, false];
    }
    if (!enabled) {
        return [k, null, true];
    }
    if (search) {
        return [k, i, true];
    }
    return [k, enabled, true];
}

function clientTimezone(dd) {
    const input = q("input", dd);
    if (input.value !== "" && !input.value.includes("UTC ??:??")) {
        return input.value;
    }
    const items = qAll(".list .item");
    const tzName = Intl.DateTimeFormat().resolvedOptions().timeZone;
    for (const item of items) {
        if (item.getAttribute("name") === tzName) {
            return itemToText(item, true);
        }
    }
    return "";
}

function parseDropdown(dd, search) {
    
    const f = findAncestor(".field", dd);
    const input = q("input", dd);
    
    // Populate dropdown list if necessary.
    const actions = input.dataset.action.split(",");
    for (let a of actions) {
        a = a.trim();
        if (!a.startsWith("populateDropdownFromTextInputs")) {
            continue;
        }
        const arg = a.split("[")[1].slice(0, -1);
        populateDropdownFromTextInputs({target: input}, arg);
    }
    
    let v = input.value;
    const k = search ? input.dataset.id : elemName(input);
    const items = qAll(".list .item", dd);
    const opt = isOptional(input, search);

    const vSet = window[f.dataset.valueSet];
    if (vSet) {
        v = vSet(dd);
    }
    
    if (opt && !v) {
        return [null, null, true];
    }
    
    let seenVal;
    let seenIdx;
    v = v.toLowerCase();
    for (let i = 0; i < items.length; i++) {
        const text = itemToText(items[i], false);
        if (text === v) {
            /*
                If the list contents is known and static we submit
                the item name as the value. Otherwise, we assume
                it's dynamically generated from user input.
            */
            if (items[i].hasAttribute("name")) {
                v = items[i].getAttribute("name");
            }
            seenVal = true;
            seenIdx = items[i].dataset.idx ? items[i].dataset.idx : i;
        }
    }
    
    if (!opt && !v) {
        requiredError(dd, search);
        return [null, null, false];
    }
    
    if (!seenVal) {
        const msg = "Value in textfield does not match any of the items in the list.";
        addError(dd, msg);
        return [null, null, false];
    }

    if (search) {
        const vMod = window[f.dataset.valueModify];
        if (vMod) {
            seenIdx = vMod(seenIdx);
        }
        return [k, seenIdx, true];
    }
    return [k, v, true];
}

function parseTime(t, search) {

    const hr = q(".inputs .hour", t);
    const min = q(".inputs .min", t);
    const am = q(".meridiem input", t);

    const k = search ? t.dataset.id : elemName(t);
    const opt = isOptional(t, search);

    if (hr.value === "00") {
        if (opt) {
            return [null, null, true];
        } else {
            requiredError(t, search);
            return [null, null, false];
        }
    }

    if (!search) {
        let h = parseInt(hr.value);
        if (am.checked && h === 12) {
            h = 0;
        }
        if (!am.checked && h !== 12) {
            h += 12;
        }
        h = (h * 60) * 60;
        const m = parseInt(min.value) * 60;
        return [k, h + m, true];
    }

    v = mapCompactTime(hr.value, min.value, am.checked);
    if (v === null) {
        throw "Couldn't map time value to compact query format."
    }
    return [k, v, true];
}

function mapCompactTime(hh, mm, isAm) {

    hh = parseInt(hh);
    mm = parseInt(mm);

    if (!isAm && hh >= 1 && hh <= 10 && mm === 0) {
        if (hh === 10) {
            return 0;
        }
        return hh;
    }

    let next = 10;
    if (mm % 15 === 0) {
        for (let i = 1; i < 13; i++) {
            for (let j = 0; j < 4; j++) {
                thisMM = j * 15;
                if (isAm && hh === i && thisMM === mm) {
                    return next;
                }
                next++;
                if (i > 10 || j > 0) {
                    if (!isAm && hh === i && thisMM === mm) {
                        return next;
                    }
                    next++;
                }
            }
        }
    }

    next = 96;
    let leadingZeroes = 0;
    let seen100 = false;
    let seen10 = false;
    for (let i = 1; i < 13; i++) {
        for (let j = 1; j < 60; j++) {
            if (j % 15 === 0) {
                continue;
            }
            for (const thisIsAm of [true, false]) {
                if (!seen100 && next === 100) {
                    next = -9;
                    leadingZeroes = 2;
                    seen100 = true;
                }
                if (!seen10 && next === 10) {
                    next = -99;
                    leadingZeroes = 3;
                    seen10 = true;
                }
                if (i === hh && j === mm && thisIsAm === isAm) {
                    return addLeadingZeroes(next, leadingZeroes);
                }
                next++;
            }
        }
    }

    return null;
}

function addLeadingZeroes(n, count) {
    const isNeg = n < 0;
    if (isNeg) {
        count--;
        if (count < 0) {
            count = 0;
        }
    }
    let s = ("" + Math.abs(n)).padStart(count, "0");
    if (isNeg) {
        s = "-" + s;
    }
    return s;
}

const monthNames = [
    "January",
    "February",
    "March",
    "April",
    "May",
    "June",
    "July",
    "August",
    "September",
    "October",
    "November",
    "December",
];

function parseCalendar(cal, search) {

    const val = q(".value", cal);
    const k = search ? val.dataset.id : elemName(val);
    const v = val.textContent;
    const opt = isOptional(val, search);

    if (v === "") {
        if (opt) {
            return [null, null, true];
        } else {
            requiredError(cal, search);
            return [null, null, false];
        }
    }

    if (v === "Present" && !search) {
        return [k, presentThreshold, true];
    }

    // Seconds.
    let unix;
    if (v === "Present") {
        unix = 0;
    } else {
        if (cal.dataset.type === "date") {
            let [mm, dd, yyyy] = v.split(" ");
            yyyy = parseInt(yyyy);
            mm = monthNames.indexOf(mm)
            dd = parseInt(dd.slice(0, -1))
            unix = Date.UTC(yyyy, mm, dd);
        } else {
            const [mm, yyyy] = v.split(" ");
            unix = Date.UTC(parseInt(yyyy), monthNames.indexOf(mm));
        }
    }
    unix /= 1000;

    if (search) {
        // Days.
        unix = ((unix / 60) / 60) / 24
        return [k, Math.round(unix) , true]
    }

    return [k, Math.round(unix), true];
}

function parseImage(img, search) {
    
    const input = q("input", img);
    const ff = input.files;
    const k = elemName(input);
    const opt = isOptional(input, search);
    const field = findAncestor(".field", img);
    
    if (ff.length === 0) {
        let imgName = input.dataset.img;
        if (imgName) {
            // We only send the filename.
            parts = imgName.split("/");
            imgName = parts[parts.length-1];
            return [k, imgName, true];
        }
    }

    if (!opt && ff.length === 0) {
        requiredError(img, search);
        return [null, null, false];
    }

    if (ff.length > 0 && field.hasAttribute("data-max")) {
        // Note: max is in kilobytes, not bytes.
        const max = parseInt(field.dataset.max);
        if (ff[0].size > max*1024) {
            addError(img, "Image exceeds maximum file size.");
            return [null, null, false];
        }
    }

    return [k, ff[0], true];
}

function parseEditor(editor, search) {
    
    const input = q(".input", editor);
    const k = elemName(input);
    const opt = isOptional(input, search);
    const paras = [];
    const structuredParas = [];
    lineariseParagraphs(input, paras);
    
    let text = "";
    for (const p of paras) {

        let pText = ""
        let tag = tagName(p);
        if (tag === "li") {
            tag = tagName(p.parentNode);
        }
        
        const spans = [];
        
        for (const span of Array.from(p.children)) {

            const spanData = {};
            if (span.dataset.link) {
                spanData.link = span.dataset.link;
            }
            const format = [];
            for (const cls of Array.from(span.classList)) {
                if (cls === "a") {
                    continue;
                }
                format.push(cls);
            }
            if (format.length > 0) {
                spanData.format = format;
            }
            /*
                This call to the replace method is a temporary fix.
                Sometimes a zero-width space character ("\u200b")
                gets trapped (especially at the end of a paragraph)
                in the text and causes a server error. We filter any
                such character out here to avoid the user receiving
                a confusing error that they can't fix.

                TODO: The editor honestly needs a re-write. Perhaps
                consider porting over piecetable from Go.
            */
            spanData.text = span.textContent.replace(/[\u200b]/, "");

            // Skip empty spans.
            if (spanData.text.length === 0) {
                continue;
            }
            
            pText += spanData.text;
            spans.push(spanData);
        }
        
        // Skip empty paragraphs.
        if (pText.length === 0) {
            continue;
        }
        text += pText
        structuredParas.push({kind: tag, span: spans});
    }
    
    const err = [null, null, false];
    text = text.trim();
    if (!opt && text === "") {
        requiredError(editor, search);
        return err;
    }
    if (text === "") {
        return [null, null, true];
    }
    if (ctrlChar(editor, text, search)) {
        return err;
    }
    if (textInvalidLength(editor, text, search)) {
        return err;
    }
    
    return [k, structuredParas, true];
}

function parseTagger(tagger, search) {
    
    const opt = isOptional(tagger, search);
    const k = elemName(tagger);
    const vv = [];
    const err = [null, null, false];
    const tags = qAll(".tag", tagger);
    
    const add = tagger.dataset.add ? parseInt(tagger.dataset.add) : false;
    if (add && tags.length > add) {
        const f = findAncestor(".field", tagger);
        addError(f, `Field cannot contain more than ${add} tags.`);
        return err;
    }
    for (const tag of tags) {
        const v = tag.textContent.trim().toLowerCase();
        if (ctrlChar(tagger, v, search)) {
            return err;
        }
        if (textInvalidLength(tagger, v, search)) {
            return err;
        }
        vv.push(v);
    }
    if (!opt && vv.length === 0) {
        requiredError(tagger, search);
        return err;
    }
    if (vv.length === 0) {
        return [null, null, true];
    }
    return [k, vv, true];
}

function invalidAdders(form, search) {
    
    if (search) {
        return;
    }
    
    let invalid = false;
    let adders = qAll(".adder", form)
    adders = adders.filter(a => !findAncestor(".button", a));
    
    for (const a of adders) {
        
        const min = a.dataset.addMin ? parseInt(a.dataset.addMin) : false;
        const max = a.dataset.add ? parseInt(a.dataset.add) : false;
        const n = Array.from(a.children).slice(0, -1).length;
        
        let atLeast = "";
        if (min && max && min === max) {
            atLeast = "at least";
        }
        if (min && n < min) {
            addError(a, `Must add ${atLeast} ${min} instances of this field.`);
            invalid = true;
        }
    }
    
    return invalid;
}

function requiredError(elem, search) {
    if (search) {
        return;
    }
    addError(elem, "This field is required.");
}

function addError(elem, msg) {
    const f = findAncestor(".field", elem);
    const errors = q(".errors", f);
    const div = document.createElement("div");
    div.textContent = msg;
    errors.appendChild(div);
}

function addFieldError(fb, name, msg) {
    if (fb[name]) {
        fb[name].push(msg);
    } else {
        fb[name] = [msg];
    }
}

function isOptional(elem, search) {
    return elem.dataset.optional === "true" || search;
}

function clearErrors(form) {
    const errs = qAll(".errors", form);
    for (const err of errs) {
        err.innerHTML = "";
    }
}

function addFeedback(fb, form) {
    
    for (let field in fb) {
        
        const errDest = q(".error_" + field, form);
        errDest.innerHTML = "";
        
        for (let i = 0; i < fb[field].length; i++) {
            
            const err = document.createElement("div");
            const msg = fb[field][i];
            err.textContent = msg;
            
            const errDest = q(".error_" + field, form);
            errDest.appendChild(err);
        }
    }
}