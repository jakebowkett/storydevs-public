
function keys(o) {
    return Object.keys(o);
}

/*
StoryDevs automatically initialises event handlers on
elements containing "data-action" attributes. By default
it will add it as a click event. For example:
    
    <div data-action="doThing"></div>
    
Will cause the div to call doThing on click with the
event object as a parameter.

The event type can be specified by adding a "data-evt"
attribute. For example:

    <div data-action="doThing" data-evt="mouseover"></div>

Means that doThing will now execute on mouseover.

Lastly, multiple events can be added in this manner:

    <div
        data-action="
            doThing,
            sayHello,
            sayGoodbye,
        "
        data-evt="
            mouseover,
            touchstart,
            keydown,
        "
    ></div>
    
The doThing handler will execute on mouseover, sayHello
on touchstart, and sayGoodbye on keydown.
*/
function initActionables(elem) {
    
    let actionables = qAll("[data-action]", elem);
    let inits = qAll("[data-init]", elem);
    
    for (const init of inits) {
        window[init.dataset.init](init);
    }
    
    for (const actionable of actionables) {
        
        const cutset = ", \n\r\t"
        let actions = trim(actionable.dataset.action, cutset).split(/,\s*/);
        let eventTypes = [];
        
        if (actionable.dataset.evt) {
            eventTypes = trim(actionable.dataset.evt, cutset).split(/,\s*/);
        }
        
        for (let i = 0; i < actions.length; i++) {
            
            let action = actions[i];
            
            let args;
            if (action.includes("[")) {
                action = action.slice(0, -1);
                const parts = action.split("[");
                action = parts[0];
                args = parts[1].split(";");
            }
            
            let func = window[action];
            let eventType;

            if (func === undefined) {
                throw `function "${action}" does not exist during initActionables`;
            }
            
            if (eventTypes.length > i) {
                eventType = eventTypes[i];
            } else {
                eventType = "click";
            }

            /*
                Each handler is given a generated name based on its
                function name and its arguments. It's important to
                include the arguments otherwise actions with the same
                name will potentially receive stale arguments. The
                generated name must be deterministic so that it can
                be removed in the future.
            */
            let handlerName = "Handler" + action + (args ? args.join("_") : "");
            if (!window[handlerName]) {
                window[handlerName] = (e) => {
                    if (args === undefined) {
                        func(e);
                    } else {
                        func(e, ...args);
                    }
                };
            }
            let handler = window[handlerName];

            actionable.removeEventListener(eventType, handler);
            actionable.addEventListener(eventType, handler);
        }
    }
}

function prependChild(newNode, parentNode) {
    parentNode.insertAdjacentElement("afterbegin", newNode);
}

function insertBefore(newNode, refNode) {
    refNode.insertAdjacentElement("beforebegin", newNode);
}

function insertAfter(newNode, refNode) {
    refNode.insertAdjacentElement("afterend", newNode);
}

function replaceNode(newNode, oldNode) {
    oldNode.parentNode.replaceChild(newNode, oldNode);
}

function removeNode(node) {
    node.parentNode.removeChild(node);
}

// We trim the string to prevent any whitespace nodes being inserted.
function strToElem(s) {
    const template = document.createElement('template');
    s = s.trim(); 
    template.innerHTML = s;
    return template.content.firstChild;
}
function strToElems(s) {
    const template = document.createElement('template');
    s = s.trim(); 
    template.innerHTML = s;
    return Array.from(template.content.children);
}

function elemEmpty(elem) {
    return elem.children.length === 0;
}

function elemType(elem) {
    return elem.getAttribute("type");
}

function elemName(elem) {
    if (elem.dataset.to) {
        return elem.dataset.to;
    }
    if (elem.dataset.name) {
        return elem.dataset.name;
    }
    return elem.getAttribute("name");
}

function tagName(elem) {
    return elem.tagName.toLowerCase();
}

function contains(a, s) {
    if (a.indexOf(s) === -1) {
        return false;
    }
    return true;
}

// Note that getComputedStyle returns pixel values for things like width
// even if they were originally defined as ems, percentages, etc.
function css(elem, property) {
    // Note that elem could be a selector string or a
    // reference to an actual element.
    if (typeof elem === 'string') elem = q(elem);
    let style = window.getComputedStyle(elem);
    return style.getPropertyValue(property);
}

function cssPseudo(elem, pseudo, property) {
    if (typeof elem === 'string') elem = q(elem);
    let style = window.getComputedStyle(elem, pseudo);
    return style.getPropertyValue(property);
}

function cssPx(elem, property) {
    let n = css(elem, property).slice(0, -2);
    n = parseFloat(n);
    return n;
}

function getTransitionDuration(elem) {
    let n = css(elem, 'transition-duration').slice(0, -1);
    n = parseFloat(n) * 1000;
    return n;
}

function runeAt(s, i) {
    return String.fromCodePoint(s.codePointAt(i));
}

function len(s) {
    let i = 0;
    for (const r of s) {
        i++;
    }
    return i;
}

function clamp(n, min, max) {
    if (n < min) {
        return min;
    }
    if (n > max) {
        return max;
    }
    return n;
}

/*
    Key/val pairs in activeAnimations will
    be objects like this:
    
        id: {
            elem: elemRef,
            prop: "scrollTop",
            duration: 400, // milliseconds
            target: 500, // pixels
        }
        
    If a new animation is started and one
    is already ongoing for the same element
    and property, the ongoing one will be
    cancelled and the new one will begin.
*/
var activeAnimations = {};
var animCount = 0;

function animate(elem, prop, duration, target, timingFunc, valueFunc, cb) {

    valueFunc = valueFunc === undefined ? (v) => { return v } : valueFunc;

    // Convert seconds to milliseconds.
    if (duration <= 0) {
        throw "duration must be a positive, non-zero number.";
    }
    duration *= 1000;

    // See activeAnimations comment above.
    for (const currentId in activeAnimations) {
        const aa = activeAnimations[currentId];
        if (aa.elem === elem && aa.prop === prop) {
            aa.duration = duration;
            aa.target = target;
            return;
        }
    }
    
    const id = animCount++;
    activeAnimations[id] = {
        elem: elem,
        prop: prop,
        duration: duration,
        target: target,
    };
    const aa = activeAnimations[id];
    
    // Determine object whose property we'll be updating. It either
    // belongs to the element itself or the element's style object.
    const obj = elem[prop] === undefined ? elem.style : elem;

    // Init some vars before the animation loop.
    let start = performance.now();
    let current = obj[prop];

    const type = typeof current;
    switch (type) {
    case "number":
        break;
    case "string":
        const matches = current.match(/-?\d+(\.\d+)?/g);
        if (matches.length !== 1) {
            throw "Could not extract exactly one number from current.";
        }
        current = parseFloat(matches[0]);
        break;
    default:
        throw `Expected current to be of type number or string, got ${type}`;
        return;
    }

    // Start animation.
    function anim(timestamp) {

        const elapsed = timestamp - start;
        const change = aa.target - current;
        const progress = elapsed / aa.duration;
        const newValue = change * timingFunc(progress) + current;
        obj[prop] = valueFunc(newValue);

        // Delete this animation's entry and execute any supplied callback.
        if (elapsed >= aa.duration) {
            delete activeAnimations[id];
            if (cb) {
                cb();
            }
            return;
        }

        // Recurse if we're still animating.
        requestAnimationFrame(anim);
    }
    
    requestAnimationFrame(anim);
}

function timingLinear(t) {
    return t;
}

function timingEaseOut(t) { // cubic
    return 1 - Math.pow(1 - t, 3);
}

// function timingEaseOut(t) { // cubic
//     return t<.5 ? 4*t*t*t : t;
// }

function timingEase(t) { // cubic
    return t<.5 ? 4*t*t*t : (t-1)*(2*t-2)*(2*t-2)+1;
}

function makeSet(ss) {
    const seen = {};
    const set = [];
    for (const s of ss) {
        if (seen[s] === undefined) {
            set.push(s);
            seen[s] = true;
        }
    }
    return set;
}

function trim(s, cutset) {
    s = trimPrefix(s, cutset);
    s = trimSuffix(s, cutset);
    return s;
}

function trimPrefix(s, cutset) {
    const set = cutset.split("");
    let start = 0;
    for (const c of s) {
        if (!set.includes(c)) {
            break;
        }
        start++;
    }
    return s.slice(start);
}

function trimSuffix(s, cutset) {
    const set = cutset.split("");
    let end = s.length;
    for (let i = s.length-1; i >= 0; i--) {
        const c = s[i];
        if (!set.includes(c)) {
            break;
        }
        end--;
    }
    return s.slice(0, end);
}

function urlToParts(url) {
    
    url = trim(url, "/");
    
    let parts = url.split("?");
    let path;
    let query;
    path = parts[0];
    if (parts.length === 2) {
        query = parts[1];
    }
    
    parts = url.split("#");
    let anchor = "";
    path = parts[0];
    if (parts.length === 2) {
        anchor = parts[1];
    }
    
    anchor = anchor ? "#" + anchor : null;
    query  = query  ? "?" + query  : null;
    
    return {
        pathSeg: path.split("/"),
        path: "/" + path,
        anchor: anchor,
        query: query,
    };
}

function copyObfuscated(e) {
    e.preventDefault();
    const item = e.currentTarget;
    const value = q(".value", item);
    value.classList.add("copied");
    setTimeout(function() {
        value.classList.remove("copied");
    }, 1000 * 1.1);
    const letters = value.children[0].children;
    let s = "";
    for (let i = 0; i < letters.length; i++) {
        if (i % 2 == 1) {
            continue;
        }
        s += letters[i].textContent;
    }
    toClipboard(s);
}

function toClipboard(s) {
    navigator.clipboard.writeText(s).then(function() {
        // success
    }, function() {
        // failure
    });
}

