
function get(path, callback) {
    request("GET", path, null, callback);
}
function post(path, data, callback) {
    request("POST", path, data, callback);
}
function put(path, data, callback) {
    request("PUT", path, data, callback);
}
function patch(path, data, callback) {
    request("PATCH", path, data, callback);
}
function del(path, callback) {
    request("DELETE", path, null, callback);
}

function request(method, path, data, callback) {
    
    if (!contains(["GET", "POST", "PUT", "PATCH", "DELETE"], method)) {
        throw "Invalid HTTP method in call to request.";
    }
    
    const xhr = new XMLHttpRequest();
    xhr.open(method, path, true);

    xhr.onreadystatechange = function() {

        if (xhr.readyState !== XMLHttpRequest.DONE) {
            return;
        }
        
        if (xhr.status === 0) {
            
            const options = {method: "HEAD", mode: "no-cors"};
            fetch("https://www.google.com", options)
            
            .then(r => {
                const title = "Server Error"
                const msg = "StoryDevs appears to be down."
                showNotification(title, msg, "warn");
                callback(new Error(title));
            })
            
            .catch(e => {
                const title = "Network Error"
                const msg = "Check your internet connection."
                showNotification(title, msg, "warn");
                callback(new Error(title));
            });
            
            return;
        }
        
        const contentType = xhr.getResponseHeader("Content-Type");
        
        let res = xhr.response;
        if (contentType && contentType.indexOf("json") !== -1) {
            res = JSON.parse(res);
        }
        
        if (xhr.status >= 400) {
            const err = {
                status: xhr.status,
                statusText: xhr.statusText,
                msg: res,
            };
            showNotification(xhr.statusText, res, "error");
            callback(err);
            return;
        }

        callback(null, res);
    };
    
    if (data) {
        if (data instanceof FormData) {
            xhr.send(data);
        } else {
            xhr.send(JSON.stringify(data));
        }
    } else {
        xhr.send();
    }
}

function notificationVisible(note) {
    return !note.classList.contains("hidden");
}

function showNotification(titleText, msgText, kind) {
    
    const note = q("#notification");
    const title = q(".title", note);
    const msg = q(".msg", note);
    const kinds = ["error", "warn", "info", "success"];

    if (!kind) {
        kind = "info";
    }

    let delay = 0;
    if (notificationVisible(note)) {
        dismissNotification();
        delay = getTransitionDuration(note);
    }
    
    setTimeout(function() {
        title.textContent = titleText;
        msg.textContent = msgText;
        for (const k of kinds) {
            note.classList.remove(k);
        }
        note.classList.add(kind);
        note.classList.remove("hidden");
    }, delay);
}

function dismissNotification() {
    const note = q("#notification");
    const title = q(".title", note);
    const msg = q(".msg", note);
    note.classList.add("hidden");
    msg.textContent = "";
    title.textContent = "";
}