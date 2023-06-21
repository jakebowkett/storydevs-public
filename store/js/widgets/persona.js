
/*
TODO: displaying loading icon while request is on-going
*/
// TODO: reload persona-related stuff

/*
TODO: consider assigning a data-owner property
that allows us to query which results/resources
are owned by a given profile. Will help with
displaying/hiding edit/delete/etc buttons on a
persona switch.
*/

function showPersona(e) {
    e.preventDefault();
    showModal("persona");
}

function addPersona(res) {
    
    const adder = q(`.adder[name="personas"]`);
    const btn = q(`.adder[name="personas"] > .button`);
    
    addFieldInstance({target: btn});
    
    const instance = adder.children[adder.children.length-2];
    q(".name", instance).textContent = res.handle;
    q(".handle", instance).textContent = "@" + res.handle;
    q(".wdgt-persona", instance).dataset.slug = res.slug;

    const persBtn = q(`#persona_switcher .btn[data-action="switchPersonaFromSidebar"]`);
    const persSect = findAncestor(".section", persBtn);
    const newPers = persBtn.cloneNode(true);
    newPers.dataset.slug = res.slug
    q(".icon.avatar", newPers).innerHTML = "";
    q(".text", newPers).innerHTML = res.handle;
    persSect.appendChild(newPers);
    initActionables(persSect);
}

function switchPersonaFromAccount(e) {
    const p = e.currentTarget;
    const a = q(".avatar img", p);
    const avatar = a ? a.getAttribute("src") : null;
    switchPersona({
        slug: p.dataset.slug,
        handle: q(".handle", p).textContent,
        avatar: avatar,
    });
}

function switchPersonaFromSidebar(e) {
    const btn = e.currentTarget;
    const a = q(".avatar img", btn);
    const avatar = a ? a.getAttribute("src") : null;
    switchPersona({
        slug: btn.dataset.slug,
        handle: q(".text", btn).textContent,
        avatar: avatar,
    });
}

function switchPersona(to) {
    put("/switch/" + to.slug, null, (err) => {
        if (err) {
            log(err);
            return;
        }
        switchPersonaInterface(to);
    });
}

function switchPersonaInterface(to) {

    const wp = qAll(".wdgt-persona");
    for (const p of wp) {
        if (p.dataset.slug === to.slug) {
            p.classList.add("selected");
            continue;
        }
        p.classList.remove("selected");
    }

    q("#persona_switcher").dataset.slug = to.slug;
    q("#persona_switcher > .link").textContent = trimPrefix(to.handle, "@ ");

    const avatar = q("#persona_switcher > .avatar");
    if (to.avatar) {
        const img = document.createElement("img");
        img.setAttribute("src", to.avatar);
        avatar.innerHTML = "";
        avatar.appendChild(img);
    } else {
        avatar.innerHTML = "";
    }

    /*
        When switching personas in the account view
        we reload the browse column for non-settings
        subviews. For settings the browse column is
        constant but we do reload the detail column
        as it contains persona-specific data.
    */
    const c = context;
    if (c.view === "account") {
        if (!c.subView) {
            return;
        }
        if (c.subView !== "settings") {
            loadColumn("browse", `/account/${c.subView}`);
            return;
        }
        if (c.resource) {
            loadColumn("detail", `/account/settings/${c.resource}`);
            return;
        }
    }
}

function deletePersonaPrompt(e) {
    e.preventDefault();
    const instance = findAncestor(".instance", e.currentTarget);
    const p = q(".wdgt-persona", instance);
    const handle = q(".handle", p).textContent;
    const slug = p.dataset.slug;
    let name = q(".name", p).textContent;
    showModal("delete", (s) => {
        
        // Replace action
        s = s.replace(/deleteResource/, `deletePersona[${slug}]`);
        
        // Create temporary element we can query.
        tmp = document.createElement("div");
        tmp.innerHTML = s;
        
        // Replace header text.
        q("h2", tmp).textContent = "Delete Persona"
        
        // Replace the paragraphs' text.
        const p1 = q("form > p", tmp);
        name = `<span class="b">${name}</span>`;
        p1.innerHTML = `Are you sure you want to delete ${name} (${handle})?`;
        
        // Add another paragraph.
        const p2 = document.createElement("p");
        p2.innerHTML = (
            `Talent profiles belonging to this persona will be deleted. ` +
            `Forum and library posts will remain but your name, handle, and avatar ` +
            `will no longer be associated with them.`
        );
        insertAfter(p2, p1);
        
        // Update hidden modal to match visible one.
        q("#modal_hidden", tmp).innerHTML = q("form", tmp).outerHTML;
        
        // Return the updated modal.
        return tmp.innerHTML;
    });
}

function deletePersona(e, slug) {
    
    // res is the newly active persona, if the active one has been deleted
    del("/persona/" + slug, (err, res) => {
        
        if (err) {
            log(err);
            return;
        }
        
        to = {slug: res};

        const wp = qAll(".wdgt-persona");
        let instance;
        for (const p of wp) {
            if (p.dataset.slug === slug) {
                instance = findAncestor(".instance", p);
            }
            if (p.dataset.slug === res) {
                const a = q(".avatar img", p);
                to.handle = q(".handle", p).textContent;
                to.avatar = a ? a.getAttribute("src") : null;
            }
        }
        
        removeFieldInstance({
            target: instance,
            stopPropagation: () => {},
        });
        removeNode(q(`#persona_switcher .btn[data-slug="`+slug+`"]`));
        switchPersonaInterface(to);
    });
}
