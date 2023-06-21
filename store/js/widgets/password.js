
function forgotPassword(e) {
    e.preventDefault();
    e.stopPropagation();

    get("/forgot/partial", (err, res) => {

        if (err) {
            log(err);
            return;
        }

        const modal = q("#modal");
        emptyModal(modal);
        populateModal(modal, res);
        const inner = q(".inner", modal);

        const firstInput = q("input", inner);
        if (firstInput) {
            firstInput.focus();
        }

        init(inner);
    });
}

function passwordInput(e) {
    
    const input = e.target;
    const text = input.value;
    const password = findAncestor(".password", input);
    const weak = q(".weak");
    const okay = q(".okay");
    const strong = q(".strong");
    
    if (text.length === 0) {
        setPasswordMeter(password, null);
        return;
    }
    
    /*
        Matching the below rules automatically
        makes a password be set as weak.
    */
    let isWeak = false;
    if (text.length < 18) {
        if (text.match(/st(o|0)ryd(e|3)vs/gi)) {
            isWeak = true;
        }
        if (text.match(/p(a|@)ssw(o|0)rd/gi)) {
            isWeak = true;
        }
        if (text.match(/qwerty/gi)) {
            isWeak = true;
        }
        if (text.match(/abc/gi)) {
            isWeak = true;
        }
        if (text.match(/123/gi)) {
            isWeak = true;
        }
        if (text.match(/19\d\d/gi) || text.match(/20\d\d/)) {
            isWeak = true;
        }
        if (text.match(/(\d+)\1{1,}/gi)) { // 11, 123123, 1212, etc
            isWeak = true;
        }
        if (text.match(/(.)\1{2,}/gi)) { // aaa, $$$, etc
            isWeak = true;
        }
    }
    if (text.length < 9) {
        isWeak = true;
    }
    if (isWeak) {
        setPasswordMeter(password, "weak");
        return;
    }
    
    const number = text.match(/\d/g);
    const special = text.match(/[^\s\dA-Za-z]/g);
    
    let multiple = 0
    if (number) {
        for (const n of number) {
            multiple++;
        }
    }
    if (special) {
        for (const n of special) {
            multiple += 2;
        }
    }
    
    if (text.length < 15) {
        
        if (multiple < 3) {
            setPasswordMeter(password, "weak");
            return;
        }
        
        setPasswordMeter(password, "okay");
        return;
    }
    
    if (text.length < 22) {
        
        if (multiple < 2) {
            setPasswordMeter(password, "okay");
            return;
        }
        
        setPasswordMeter(password, "strong");
        return;
    }
    
    setPasswordMeter(password, "strong");
}

function setPasswordMeter(password, strength) {
    const meter = q(".meter", password);
    for (const child of Array.from(meter.children)) {
        child.classList.remove("on");
    }
    if (!strength) {
        return;
    }
    q("." + strength).classList.add("on");
}