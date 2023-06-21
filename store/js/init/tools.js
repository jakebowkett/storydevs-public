
function log(s) {
    console.log(s);
}

function q(selector, elem) {
    if (elem) return elem.querySelector(selector);
    return document.querySelector(selector);
}

function qAll(selector, elem) {
    if (elem) return Array.from(elem.querySelectorAll(selector));
    return Array.from(document.querySelectorAll(selector));
}

function findAncestor(selector, elem) {
    let isId, isClass, isAttr, isTag;
    switch (selector[0]) {
        case '#': isId    = true; selector = selector.slice(1);      break;
        case '.': isClass = true; selector = selector.slice(1);      break;
        case '[': isAttr  = true; selector = selector.slice(1, -1);  break;
        default : isTag   = true; selector = selector.toUpperCase(); break;
    }
    while (elem) {
        if (elem.nodeType !== Node.ELEMENT_NODE) {
            elem = elem.parentNode;
            continue;
        }
        if      (isId    && elem.id === selector)              break;
        else if (isClass && elem.classList.contains(selector)) break;
        else if (isAttr  && elem.getAttribute(selector))       break;
        else if (isTag   && elem.tagName === selector)         break;
        elem = elem.parentNode;
    }
    return elem;
}

function inRect(elem, x, y) {
    let r = elem.getBoundingClientRect();
    if (x < r.left) {
        return false
    }
    if (x > r.right) {
        return false
    }
    if (y < r.top) {
        return false
    }
    if (y > r.bottom) {
        return false
    }
    return true;
}
