
function addImage(input) {

    const files = input.files;
    const container = findAncestor(".image", input);
    const preview = q(".preview", container);
    const meta = q(".meta", container);
    const size = q(".size", meta);
    
    if (files.length === 0) {
        container.classList.remove("present");
        preview.style.backgroundImage = "";
        q(".name", meta).textContent = "";
        q(".n", size).textContent = "";
        q(".unit", size).textContent = "";
        return;
    }
    
    const file = files[0];
    const fileURL = URL.createObjectURL(file);
    const img = document.createElement("img");
    
    img.src = fileURL;
    img.onload = function() {
        window.URL.revokeObjectURL(fileURL);
    }
    preview.style.backgroundImage = 'url("' + img.src + '")';
    
    // Metadata.
    const [n, unit] = formatBytes(file.size);
    q(".name", meta).textContent = file.name ? file.name : "";
    q(".n", size).textContent = n;
    q(".unit", size).textContent = unit;
    
    container.classList.add("present");
}

function formatBytes(bytes) {
    
    if (bytes < 1000) {
        return [bytes, "B"];
    }
    
    const kb = bytes / 1024;
    if (kb < 1000) {
        return [parseInt(kb), "KB"];
    }
    
    const mb = kb / 1024;
    return [mb.toFixed(1), "MB"];
}