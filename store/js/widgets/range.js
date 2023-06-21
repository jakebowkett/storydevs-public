
var rangeMeta;

function updateRanges(elem) {
    
    for (const range of qAll(".range-wdgt", elem)) {
        
        const start = q(".start", range);
        const end = q(".end", range);
        const segNum = q(".track", range).children.length;
        const [startPos, endPos] = rangeMarkerPositions(range);
        const prop = mobile ? "top" : "left";
        start.style[prop] = ((100 / segNum) * startPos).toFixed(4) + "%";
        end.style[prop]   = ((100 / segNum) * endPos).toFixed(4)   + "%";
        
        /*
            Force rangeDown to reinitialise and
            convert percentages to pixels.
        */
        range.dataset.init = null;
    }
}

function rangeMarkerPositions(range) {
    const start = q(".start", range);
    const end = q(".end", range);
    return [parseInt(start.dataset.pos), parseInt(end.dataset.pos)];
}

function rangeInit(range) {
    
    const meta = rangeMeta;
    const start = q(".start", range);
    const end = q(".end", range);
    
    for (const elem of [start, end]) {
        
        if (mobile) {
            const per = elem.style.top.slice(0, -1);
            const px = Math.round(range.clientHeight * (per / 100));
            elem.style.top = px + "px";
            continue;
        }
        
        const per = elem.style.left.slice(0, -1);
        const px = Math.round(range.clientWidth * (per / 100));
        elem.style.left = px + "px";
    }
    
    setRangeMarkerPos(start);
    setRangeMarkerPos(end);
}

function setRangeMarkerPos(marker) {
    
    const meta = rangeMeta;
    let side;
    if (mobile) {
        side = marker.classList.contains("start") ? "top" : "bottom";
    } else {
        side = marker.classList.contains("start") ? "left" : "right";
    }
    
    let offset = q(".marker", marker).getBoundingClientRect()[side];
    offset -= meta.rangeStart;
    
    let seg = meta.segSize/2;
    let selected = meta.segNum;
    for (let i = 0; i < meta.segNum; i++) {
        if (offset < seg) {
            selected = i;
            break;
        }
        seg += meta.segSize;
    }
    
    marker.dataset.pos = selected;
}

function rangeDown(e) {

    // Hard to test mobile if mouse events are going off.
    if (mobile) {
        return;
    }

    // Ensure we're dealing with left mouse button.
    if (e.button !== 0) {
        return;
    }
    
    e.preventDefault();
    rangeGeneralStart(e);
    
    window.addEventListener("mousemove", rangeGeneralMove);
    window.addEventListener("mouseup", rangeUp);
}

function rangeGeneralStart(e) {
    
    let marker;
    if (mobile) {
        marker = findAncestor(".start", e.target);
        if (!marker) {
            marker = findAncestor(".end", e.target);
        }
    } else {
        marker = e.currentTarget;
    }
    
    const range = findAncestor(".range-wdgt", marker);
    const kind = marker.classList.contains("start") ? "start" : "end";
    const otherKind = marker.classList.contains("start") ? "end" : "start";
    const rangeRect = range.getBoundingClientRect();
    const mRect = marker.getBoundingClientRect();
    const markerN = mobile ? mRect.top : mRect.left;
    const segNum = q(".track", range).children.length;
    const mrk = q(".marker", marker);
    
    let side, rangeStart, rangeEnd;
    if (mobile) {
        rangeStart = "top";
        rangeEnd = "bottom";
        side = kind === "start" ? "top" : "bottom";
    } else {
        rangeStart = "left";
        rangeEnd = "right";
        side = kind === "start" ? "left" : "right";
    }
    const brd = cssPx(mrk, `border-${side}-width`);
    const userPos = mobile ? e.clientY : e.clientX;
    
    rangeMeta = {
        rangeStart: rangeRect[rangeStart],
        rangeEnd: rangeRect[rangeEnd],
        otherMarker: q("."+otherKind, range),
        marker: marker,
        markerSize: mobile ? mrk.offsetHeight : mrk.offsetWidth,
        markerBorder: brd,
        kind: kind,
        offset: userPos - markerN,
        segSize: (mobile ? range.clientHeight : range.clientWidth) / segNum,
        segNum: segNum,
    };
    
    if (!range.dataset.init) {
        rangeInit(range);
        range.dataset.init = true;
    }
}

function rangeGeneralMove(e) {
    
    let side, margin;
    if (mobile) {
        side = "top";
        margin = "marginTop";
    } else {
        side = "left";
        margin = "marginLeft";
    }
    
    // Clamp to edges of range element.
    const meta = rangeMeta;
    const otherPos = parseInt(meta.otherMarker.dataset.pos);
    
    let n = (mobile ? e.clientY : e.clientX) - meta.offset - meta.rangeStart;
    let rangeSize = meta.rangeEnd - meta.rangeStart;
    
    if (meta.kind === "start") {
        rangeSize -= meta.segSize * ((meta.segNum - otherPos) + 1);
        if (n <= 0) {
            n = 0;
        }
        if (n >= rangeSize) {
            n = rangeSize;
        }
    } else {
        let segSize = meta.segSize;
        if (!mobile) {
            segSize -= meta.markerSize;
            segSize += meta.markerBorder;
            rangeSize -= meta.markerSize;
        } else {
            rangeSize -= meta.markerBorder;
            n += (meta.markerSize - meta.markerBorder - 1);
        }
        segSize += meta.segSize * otherPos;
        if (n <= segSize) {
            n = segSize;
        }
        if (n >= rangeSize) {
            n = rangeSize;
        }
    }
    
    meta.marker.style[side] = n + "px";
    meta.marker.style[margin] = "0"; // disable margin while moving
}

function rangeUp(e) {
    rangeGeneralEnd(e);
    window.removeEventListener("mousemove", rangeGeneralMove);
    window.removeEventListener("mouseup", rangeUp);
}

function rangeGeneralEnd(e) {
    
    const meta = rangeMeta;
    let side, margin;
    if (mobile) {
        side = "top";
        margin = "marginTop";
    } else {
        side = "left";
        margin = "marginLeft";
    }
    const prop = meta.marker.style[side];
    
    let n = parseInt(prop.slice(0, -2));
    let seg = meta.segSize / 2;
    let selected = meta.segNum;
    for (let i = 0; i < meta.segNum; i++) {
        if (n < seg) {
            selected = i;
            break;
        }
        seg += meta.segSize;
    }
    n = selected * meta.segSize;
    
    if (selected === meta.segNum) {
        n -= meta.markerBorder;
    }

    meta.marker.style.transitionDuration = "0.15s";
    meta.marker.style[side] = n + "px";
    meta.marker.style[margin] = "";
    
    setTimeout(function() {
        
        meta.marker.style.transitionDuration = "";
    
        setRangeMarkerPos(meta.marker);
        const pos = parseInt(meta.marker.dataset.pos);
        const otherPos = parseInt(meta.otherMarker.dataset.pos);
        
        let start, end;
        if (pos < otherPos) {
            start = pos;
            end = otherPos;
        } else {
            start = otherPos;
            end = pos;
        }
        
        const range = findAncestor(".range-wdgt", meta.marker);
        const track = q(".track", range);
        
        let highlight;
        let status;
        for (let i = 0; i < meta.segNum; i++) {
            const seg = track.children[i];
            if (i === start) {
                status = seg.textContent;
                highlight = true;
            }
            if (i === end) {
                prevSeg = track.children[i-1];
                const tc = prevSeg.textContent;
                if (tc !== status) {
                    status += " – " + tc;
                }
                highlight = false;
            }
            if (highlight) {
                seg.classList.add("selected");
            } else {
                seg.classList.remove("selected");
            }
        }
        const tc = track.children[end-1].textContent;
        if (end === meta.segNum && tc !== status) {
            status += " – " + tc;
        }
        
        q(".status", range).textContent = status;
    
    }, 150 + 20);
}

function rangeTouchStart(e) {
    
    e.preventDefault();
    const touch = e.changedTouches[0];
    rangeGeneralStart(touch);
    
    e.target.removeEventListener("touchstart", rangeTouchStart);
    window.addEventListener("touchmove", rangeTouchMove);
    window.addEventListener("touchend", rangeTouchEnd);
}

function rangeTouchMove(e) {
    const touch = e.changedTouches[0];
    rangeGeneralMove(touch);
}

function rangeTouchEnd(e) {
    
    e.preventDefault();
    const touch = e.changedTouches[0];
    rangeGeneralEnd(touch);
    
    e.target.addEventListener("touchstart", rangeTouchStart);
    window.removeEventListener("touchmove", rangeTouchMove);
    window.removeEventListener("touchend", rangeTouchEnd);
}