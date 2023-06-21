
function textareaHeight(e) {
    const input = e.target;
    input.style.height = "";
    input.style.height = input.scrollHeight + "px";
}

function maxChars(e) {
    
    const input = e.target;
    const tf = findAncestor(".textfield", input);
    const field = tf ? tf : findAncestor(".textarea", input);
    const max = parseInt(input.dataset.max);
    const n = len(input.value);
    const counter = q(".len", field);
    
    counter.textContent = `${n}/${max}`;
    
    if (n > (max - max/8) && n <= max) {
        field.classList.add("warn");
    } else {
        field.classList.remove("warn");
    }
    
    if (n > max) {
        field.classList.add("error");
    } else {
        field.classList.remove("error");
    }
}
