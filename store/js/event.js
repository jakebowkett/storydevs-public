

function eventCountdown(cd) {
    
    // Clear recurring calls when the current view is unloaded.
    q("body").addEventListener("viewchange", () => {
        clearInterval(id);
    });
    
    // Countdown start values.
    const tm = q(".timer", cd);
    let d = parseInt(cd.dataset.dd);
    let h = parseInt(cd.dataset.hh);
    let m = parseInt(cd.dataset.mm);
    let s = parseInt(cd.dataset.ss);
    let id;

    let start  = parseInt(cd.dataset.unixStart);
    let finish;
    if (cd.dataset.unixFinish) {
        finish = parseInt(cd.dataset.unixFinish);
    }

    const tz = cd.dataset.timezone;
    if (tz === "local") {
        const unitHour = 60 * 60;
        const off = (new Date(start * 1000)).getTimezoneOffset() * 60;
        start += off;
        if (finish) {
            const finOff = (new Date(finish * 1000)).getTimezoneOffset() * 60;
            finish += finOff;
        }
        h += (off / unitHour)
        m += (off % unitHour)
        if (m >= 60) {
            m -= 60;
            h++;
        }
        if (m < 0) {
            m += 60;
            h--;
        }
        if (h >= 24) {
            h -= 24;
            d++;
        }
        if (h < 0) {
            h += 24;
            d--;
        }
    }

    updateCountdown = () => {
        
        // If seconds is not 0 we simply decrement it.
        if (s != 0) {
            s--;
            renderCountdown();
            return;
        }
        
        // Otherwise we have to go down a minute. If the
        // minute isn't 0 we can simply decrement that.
        s = 59;
        if (m != 0) {
            m--;
            renderCountdown();
            return;
        }
        
        // Otherwise we need to decrement the hour too.
        m = 59;
        if (h != 0) {
            h--;
            renderCountdown();
            return;
        }

        // Otherwise we need to decrement the day too.
        h = 23;
        if (d != 0) {
            d--;
            renderCountdown();
            return;
        }
        
        // If the day is zero we set everything to zero
        // because the event is active.
        d = 0;
        h = 0;
        m = 0;
        s = 0;
        renderCountdown();
        
        tm.classList.add("done");
        if (!id) {
            return;
        }
        clearInterval(id);
    }
    
    renderCountdown = () => {
           q(".days", tm).textContent = d < 10 ? "0"+d.toString() : d.toString();
          q(".hours", tm).textContent = h < 10 ? "0"+h.toString() : h.toString();
        q(".minutes", tm).textContent = m < 10 ? "0"+m.toString() : m.toString();
        q(".seconds", tm).textContent = s < 10 ? "0"+s.toString() : s.toString();
    }
    
    // Display local date and timer.
    displayDateTime(start, q(".date.start", cd));
    if (finish) {
        displayDateTime(finish, q(".date.finish", cd));
    }
    updateCountdown();
    
    // Begin decrementing timer until 00:00:00.
    id = setInterval(updateCountdown, 1000);
}

function displayDateTime(unix, target) {
    const d = new Date(unix * 1000);
    const day = new Intl.DateTimeFormat('en-AU', {
        year: 'numeric',
        month: 'long',
        weekday: 'long',
        day: 'numeric',
    }).format(d);
    const time = new Intl.DateTimeFormat('en-AU', {
        hour: 'numeric',
        minute: 'numeric',
        timeZoneName: 'short',
    }).format(d);
    q(".day",  target).textContent = day + ",";
    q(".time", target).textContent = time;
}
