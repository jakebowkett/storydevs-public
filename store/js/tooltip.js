
function initTooltips(elem) {
    if (mobile) {
        return;
    }
    const tips = qAll("[data-tip]", elem);
    for (const tip of tips) {
        tip.removeEventListener("mouseenter", tooltipEnter);
        tip.removeEventListener("mouseleave", tooltipDismiss);
        tip.addEventListener("mouseenter", tooltipEnter);
        tip.addEventListener("mouseleave", tooltipDismiss);
    }
}


function tooltipEnter(e) {

    /*
        This ensures that nested tooltips work correctly without
        snapping immediately to their new location when delayed.
    */
    tooltipDismiss();

    /*
        We wrap the body in a call to requestAnimationFrame to
        allow the effect of tooltipDismiss to take place. If we
        don't do this the tool tip will remain visible and snap
        to the new location even if it's supposed to appear with
        a delay.
    */
    requestAnimationFrame(() => {

        const elem = e.target;
        const tt = q("#tip");

        q(".text", tt).textContent = elem.dataset.tip;
        tt.classList.remove("hidden");

        if (elem.dataset.tipDelayed) {
            tt.classList.add("delayed");
        }

        const rect = elem.getBoundingClientRect();

        tt.style.top  = rect.bottom + "px";
        if (elem.dataset.tipClient) {
            tt.style.left = e.clientX - (tt.clientWidth / 2) + "px";
        } else {
            const elemMid = rect.left + ((rect.right - rect.left) / 2);
            tt.style.left = elemMid - (tt.clientWidth / 2) + "px";
        }

        const tipHeight = tt.clientHeight + cssPx(tt, "margin-top");
        const tipBottom = rect.bottom + tipHeight;
        if (tipBottom > window.innerHeight) {
            tt.classList.add("above");
            tt.style.top = (rect.top - tipHeight) + "px";
        }
    });
}

function tooltipDismiss() {
    const tt = q("#tip");
    tt.classList.add("hidden");
    tt.classList.remove("above");
    tt.classList.remove("delayed");
    q(".text", tt).textContent = "";
    tooltipActive = false;
}
