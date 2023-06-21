
function calendarTypeAndValue(calendar) {
    const vtc = q(".display .value", calendar).textContent;
    if (calendar.dataset.type === "calendar") {
    	switch (vtc) {
		case "":
			return "calendar-empty";
		case "Present":
			return "calendar-present";
    	}
		return "calendar-value";
    }
	switch (vtc) {
	case "":
		return "date-empty";
	case "Present":
		return "date-present";
	}
	return "date-value";
}

function calendarFocus(e) {

	const display = e.currentTarget;
    const value = q(".value", display);
    const calendar = findAncestor(".calendar", display);
    const selector = q(".selector", calendar);
    const view = q(".view", calendar);

    let yyyy;
    let mm;
    switch (calendarTypeAndValue(calendar)) {
	// Calendar format is: January, 2006
	case "calendar-empty":
        setYearAndShowMonths(calendar, new Date().getFullYear());
        break;
	case "calendar-present":
		setPresent(calendar);
        yyyy = parseInt(calendar.dataset.year);
        setYearAndShowMonths(calendar, yyyy);
        break;
	case "calendar-value":
		yyyy = parseInt(value.textContent.slice(-4));
		setYearAndShowMonths(calendar, yyyy);
		break;
	// Date format is: January 5th, 2006
	case "date-empty":
		const now = new Date();
		yyyy = now.getFullYear();
		mm = now.getMonth();
		setMonthAndShowDates(calendar, yyyy, mm);
		break;
	case "date-present":
		setPresent(calendar);
		yyyy = parseInt(calendar.dataset.year);
		mm = parseInt(calendar.dataset.month);
		setMonthAndShowDates(calendar, yyyy, mm);
        break;
	case "date-value":
		yyyy = parseInt(value.textContent.slice(-4));
		const monthName = value.textContent.split(" ", 2)[0];
		setMonthAndShowDates(calendar, yyyy, monthName);
		break;
	default:
		throw "Unknown calendar type and value combination.";
    }

    view.classList.remove("disabled");
    selector.classList.remove("hidden");
    window.addEventListener("mousedown", calendarBlur);
}

function setMonthAndShowDates(calendar, yyyy, mm) {
    const allowedRange = calendar.dataset.allowedRange;
	const dates = q(".dates", calendar);
    const frames = q(".frames", dates);
    const oldFrame = q(".frame", dates);
	const month = q(".month", calendar);
    const months = q(".months", calendar);
    const year = q(".year", calendar);
    const years = q(".years", calendar);
    const period = q(".period", calendar);
    const next = q(".next", calendar);
    const prev = q(".prev", calendar);

	if (typeof mm === "string") {
		month.textContent = mm;
        mm = monthNames.indexOf(mm);
	} else {
        month.textContent = monthNames[mm];
	}

    const now = new Date();
    const thisYear = now.getFullYear();
    const thisMonth = now.getMonth();

    next.classList.remove("disabled");
    prev.classList.remove("disabled");

    nextMonth = mm+1;
    nextYear = yyyy;
    if (nextMonth > 11) {
        nextMonth = 0
        nextYear++;
    }
    prevMonth = mm-1;
    prevYear = yyyy;
    if (prevMonth === 0) {
        prevMonth = 11
        prevYear--;
    }

    if (allowedRange === "past" &&
        (
            (nextYear > thisYear) ||
            (nextYear === thisYear && nextMonth > thisMonth)
        )
    ) {
        next.classList.add("disabled");
    }
    if (allowedRange === "future" &&
        (
            (prevYear < thisYear) ||
            (prevYear === thisYear && prevMonth < thisMonth)
        )
    ) {
        prev.classList.add("disabled");
    }

    // The ordering of these three lines matters.
	month.textContent += ` ${yyyy}`;
    const newFrame = calendarInitDates(calendar);
    frames.replaceChild(newFrame, oldFrame);

    period.classList.add("hidden");
    year.classList.add("hidden");
    years.classList.add("hidden");
    months.classList.add("hidden");
	month.classList.remove("hidden");
    dates.classList.remove("hidden");
}
function setYearAndShowMonths(calendar, yyyy) {
    const allowedRange = calendar.dataset.allowedRange;
	const year = q(".year", calendar);
    const years = q(".years", calendar);
    const period = q(".period", calendar);
	const months = q(".months", calendar);
    const frame = q(".frame", months);
    const next = q(".next", calendar);
    const prev = q(".prev", calendar);

    const now = new Date();
    const thisYear = now.getFullYear();

    next.classList.remove("disabled");
    prev.classList.remove("disabled");
    nextYear = yyyy+1;
    prevYear = yyyy-1;
    if (allowedRange === "past" && nextYear > thisYear) {
        next.classList.add("disabled");
    }
    if (allowedRange === "future" && prevYear < thisYear) {
        prev.classList.add("disabled");
    }

    // The ordering of these two lines matters.
    year.textContent = yyyy;
    calendarInitMonths(calendar);

    months.classList.remove("hidden");
    year.classList.remove("hidden");
    years.classList.add("hidden");
    period.classList.add("hidden");
    if (calendar.dataset.type === "date") {
        q(".month", calendar).classList.add("hidden");
        q(".dates", calendar).classList.add("hidden");
    }
}
function setPresent(calendar) {
	const now = new Date();
	const yyyy = now.getFullYear();
	const mm = now.getMonth();
	const dd = now.getDate();
    calendar.dataset.year = yyyy;
    calendar.dataset.month = mm;
    calendar.dataset.date = dd;
}

function calendarBlur(e) {
	const thisCalendar = findAncestor(".calendar", e.target);
    let focusedCalendar;
    let focusedSelector;
    for (const s of qAll(".calendar .selector")) {
    	if (!s.classList.contains("hidden")) {
    		focusedCalendar = findAncestor(".calendar", s);
			focusedSelector = s;
			break;
    	}
    }
    if (thisCalendar === focusedCalendar) {
    	return;
    }
	focusedSelector.classList.add("hidden");
	window.removeEventListener("mousedown", calendarBlur);
}

/*
This function should be called to dismiss the calendar
when selecting a value would result in it closing its
selector dropdown.
*/
function calendarDismiss(calendar) {
    q(".selector", calendar).classList.add("hidden");
    q(".years", calendar).classList.add("hidden");
    q(".year", calendar).classList.add("hidden");
    q(".period", calendar).classList.add("hidden");
    q(".months", calendar).classList.add("hidden");
    q(".month", calendar).classList.add("hidden");
    q(".beneath", calendar).classList.remove("hidden");
    if (calendar.dataset.type === "date") {
        q(".dates", calendar).classList.add("hidden");
    }
	window.removeEventListener("mousedown", calendarBlur);
}

function calendarClear(e) {
	const calendar = findAncestor(".calendar", e.currentTarget);
	const beneath = q(".beneath", calendar);
	const value = q(".display .value", calendar);
	value.textContent = "";
	beneath.classList.add("hidden");
	delete calendar.dataset.year;
	delete calendar.dataset.month;
	delete calendar.dataset.date;
}

function calendarInitDates(calendar) {

    const allowedRange = calendar.dataset.allowedRange;
    const month = q(".month", calendar);
    const monthName = month.textContent.split(" ", 2)[0];
    const viewYear = parseInt(month.textContent.slice(-4));
    const viewMonth = monthNames.indexOf(monthName);
    const viewDayCount = new Date(viewYear, viewMonth+1, 0).getDate();
    const viewDayCountPrev = new Date(viewYear, viewMonth, 0).getDate();

    let viewFirstDay = new Date(viewYear, viewMonth, 1).getDay() - 1;
    if (viewFirstDay < 0) {
        viewFirstDay = 6;
    }

    const list = [];
    const now = new Date();
    const thisDate = now.getDate();
    const thisMonth = now.getMonth();
    const thisYear = now.getFullYear();

    let setDate;
    let setMonth;
    let setYear;
    if (calendar.dataset.date)  setDate  = parseInt(calendar.dataset.date);
    if (calendar.dataset.month) setMonth = parseInt(calendar.dataset.month);
    if (calendar.dataset.year)  setYear  = parseInt(calendar.dataset.year);

    for (let i = 0; i < 42; i++) {
        const div = document.createElement("div");
        if (i < viewFirstDay) {
            div.textContent = (viewDayCountPrev - (viewFirstDay-1)) + i;
            div.classList.add("disabled");
            list.push(div);
            continue;
        }
        if (i >= (viewFirstDay + viewDayCount)) {
            div.textContent = i - (viewFirstDay + viewDayCount) + 1;
            div.classList.add("disabled");
            list.push(div);
            continue;
        }
        const viewDate = (i+1)-viewFirstDay;
        div.textContent = viewDate;
        if (setDate && viewDate === setDate && viewMonth === setMonth && viewYear === setYear) {
            div.classList.add("selected");
        }
        if (allowedRange === "past" &&
            (
                (viewYear > thisYear) ||
                (viewYear === thisYear && viewMonth > thisMonth) ||
                (viewYear === thisYear && viewMonth === thisMonth && viewDate > thisDate)
            )
        ) {
            div.classList.add("disabled");
        }
        if (allowedRange === "future" &&
            (
                (viewYear < thisYear) ||
                (viewYear === thisYear && viewMonth < thisMonth) ||
                (viewYear === thisYear && viewMonth === thisMonth && viewDate < thisDate)
            )
        ) {
            div.classList.add("disabled");
        }
        list.push(div);
    }

    const frame = document.createElement("div");
    frame.classList.add("frame");
    frame.append(...list);
    return frame;
}

function calendarInitMonths(calendar) {

    const allowedRange = calendar.dataset.allowedRange;
    const year = q(".year", calendar);
    const months = q(".months", calendar);
    const frame = q(".frame", months);

    const elemYear = parseInt(year.textContent);

    const list = [];
    const now = new Date();
    const thisMonth = now.getMonth();
    const thisYear = now.getFullYear();

    let setMonth;
    let setYear;
    if (calendar.dataset.month) setMonth = parseInt(calendar.dataset.month);
    if (calendar.dataset.year)  setYear  = parseInt(calendar.dataset.year);

    for (let i = 0; i < 12; i++) {
        const div = document.createElement("div");
        const elemMonth = i;
        div.textContent = monthNames[i].slice(0, 3);
        if (setMonth && elemMonth == setMonth && elemYear === setYear) {
            div.classList.add("selected");
        }
        if (allowedRange === "past" && elemMonth > thisMonth && elemYear >= thisYear) {
            div.classList.add("disabled");
        }
        if (allowedRange === "future" && elemMonth < thisMonth && elemYear <= thisYear) {
            div.classList.add("disabled");
        }
        list.push(div);
    }

    frame.innerHTML = "";
    frame.append(...list);
}

function yearRange(from, to) {
    return from + " – " + to;
}

function updateMonths(calendar) {
    const frame = q(".months .frame", calendar);
    calendarMonthsFrame(calendar, frame);
}

/*
    The frame parameter may or may not be part of the calendar
    subtree. Don't write code assuming it is -- e.g. calling
    findAncestor with frame.
*/
function calendarMonthsFrame(calendar, frame) {

    const allowedRange = calendar.dataset.allowedRange;

    // The year we're currently viewing.
    const year = q(".year", calendar);
    const viewYear = parseInt(year.textContent);

    // The year and month we're living in right now.
    const now = new Date();
    const thisYear = now.getFullYear();
    const thisMonth = now.getMonth();

    // The year/month the widget is set to, if any.
    let setMonth;
    let setYear;
    if (calendar.dataset.month) setMonth = parseInt(calendar.dataset.month);
    if (calendar.dataset.year)  setYear  = parseInt(calendar.dataset.year);

    for (let i = 0; i < frame.children.length; i++) {
        const month = frame.children[i];
        const viewMonth = i;
        if (allowedRange === "past") {
            if (viewYear > thisYear || viewYear === thisYear && viewMonth > thisMonth) {
                month.classList.add("disabled");
            } else {
                month.classList.remove("disabled");
            }
        }
        if (allowedRange === "future") {
            if (viewYear < thisYear || viewYear === thisYear && viewMonth < thisMonth) {
                month.classList.add("disabled");
            } else {
                month.classList.remove("disabled");
            }
        }
        if (setMonth && setYear === viewYear && setMonth === viewMonth) {
            month.classList.add("selected");
        } else {
            month.classList.remove("selected");
        }
    }
}

function calendarYearsFrame(oldFrame, newFrame, prev) {

    const calendar = findAncestor(".calendar", oldFrame);
    const allowedRange = calendar.dataset.allowedRange;
    const thisYear = new Date().getFullYear();

    let setYear;
    if (calendar.dataset.year) setYear = parseInt(calendar.dataset.year);

    if (prev) {
        const yyyy = parseInt(oldFrame.children[0].textContent);
        for (let i = 0; i < 12; i++) {
            const elem = newFrame.children[i];
            const viewYear = yyyy-(12-i);
            elem.textContent = viewYear;
            elem.classList.remove("disabled");
            elem.classList.remove("selected");
            if (allowedRange === "past" && viewYear > thisYear) {
                elem.classList.add("disabled");
            }
            if (allowedRange === "future" && viewYear < thisYear) {
                elem.classList.add("disabled");
            }
            if (viewYear === setYear) {
                elem.classList.add("selected");
            }
        }
        return;
    }

    const last = oldFrame.children.length-1;
    const yyyy = parseInt(oldFrame.children[last].textContent);
    for (let i = 0; i < 12; i++) {
        const elem = newFrame.children[i];
        viewYear = yyyy+(i+1);
        elem.textContent = viewYear;
        elem.classList.remove("disabled");
        elem.classList.remove("selected");
        if (allowedRange === "past" && viewYear > thisYear) {
            elem.classList.add("disabled");
        }
        if (allowedRange === "future" && viewYear < thisYear) {
            elem.classList.add("disabled");
        }
        if (viewYear === setYear) {
            elem.classList.add("selected");
        }
    }
}

function calendarSetDate(e) {

    const elem = e.target;
    if (findAncestor(".days", elem)) {
        return;
    }
    if (elem.classList.contains("disabled")) {
        return;
    }

    /*
        If a user began clicking on one date and lets up
        on another the target element will be the element
        with the highest z-order relative to the element
        with the event handler and which ultimately contains
        the target itself.
    */
    if (elem.classList.contains("dates") || elem.classList.contains("frame")) {
        return;
    }
    const dates = e.currentTarget;
    const dd = elem.textContent;
    const calendar = findAncestor(".calendar", elem);
    const value = q(".value", calendar);
    const month = q(".month", calendar);
    const selector = q(".selector", calendar);
    const beneath = q(".beneath", calendar);

    const mmStr = month.textContent.split(" ", 2)[0];
    const yyyy = month.textContent.slice(-4);

    value.textContent = `${mmStr} ${dd}, ${yyyy}`;
    calendar.dataset.year = yyyy;
    calendar.dataset.month = monthNames.indexOf(mmStr);
    calendar.dataset.date = dd;

    calendarDismiss(calendar);
}

function calendarSetMonth(e) {

    const elem = e.target;

    // See comment in calendarSetDate for similar condition's comment.
    if (elem.classList.contains("frame")) {
        return;
    }

    const months = e.currentTarget;
    const frame = q(".frame", months);
    const calendar = findAncestor(".calendar", elem);
    const value = q(".value", calendar);
    const year = q(".year", calendar);

    let mm;
    for (let i = 0; i < 12; i++) {
        if (frame.children[i] === elem) {
            mm = i;
            break;
        }
    }

    // Transition to dates selector.
    if (calendar.dataset.type === "date") {
        setMonthAndShowDates(calendar, parseInt(year.textContent), mm);
        return;
    }

    // Otherwise set the calendar value and dismiss it.
    value.textContent = `${monthNames[mm]} ${year.textContent}`;
    calendar.dataset.year = year.textContent;
    calendar.dataset.month = mm;
    calendarDismiss(calendar);
}

function calendarSetYear(e) {

    const elem = e.target;

    // See comment in calendarSetDate for similar condition's comment.
    if (elem.classList.contains("frame")) {
        return;
    }

    const calendar = findAncestor(".calendar", elem);
    const view = q(".view", calendar);
    view.classList.remove("disabled");
    setYearAndShowMonths(calendar, parseInt(elem.textContent));
}

function calendarSetPresent(e) {
	const present = e.currentTarget;
    const calendar = findAncestor(".calendar", present);
    const selector = q(".selector", calendar);
    const value = q(".value", calendar);
    const beneath = q(".beneath", calendar);
    value.textContent = "Present";
    calendarDismiss(calendar);
}

function calendarZoomOut(e) {

    const calendar = findAncestor(".calendar", e.currentTarget);
    const dates = q(".dates", calendar);
    const months = q(".months", calendar);
    const years = q(".months", calendar);
    const allowedRange = calendar.dataset.allowedRange;
    const isDateType = calendar.dataset.type === "date";
    const next = q(".next", calendar);
    const prev = q(".prev", calendar);

    // Show months.
    if (isDateType && !dates.classList.contains("hidden")) {

        const months = q(".months", calendar);
        const month = q(".month", calendar);
        const year = q(".year", calendar);
        const frame = q(".frame", months);

        const list = [];
        const now = new Date();
        const thisYear = now.getFullYear();
        const thisMonth = now.getMonth();

        let setMonth;
        let setYear;
        if (calendar.dataset.month) setMonth = parseInt(calendar.dataset.month);
        if (calendar.dataset.year)  setYear  = parseInt(calendar.dataset.year);

        const viewYear = parseInt(month.textContent.slice(-4));
        if (allowedRange === "past" && viewYear+1 > thisYear) {
            next.classList.add("disabled");
        } else {
            next.classList.remove("disabled");
        }
        if (allowedRange === "future" && viewYear-1 < thisYear) {
            prev.classList.add("disabled");
        } else {
            prev.classList.remove("disabled");
        }

        for (let i = 0; i < 12; i++) {
            const div = document.createElement("div");
            const viewMonth = i;
            div.textContent = monthNames[i].slice(0, 3);
            if (setMonth && viewMonth === setMonth && viewYear === setYear) {
                div.classList.add("selected");
            }
            if (allowedRange === "past" && viewMonth > thisMonth && viewYear >= thisYear) {
                div.classList.add("disabled");
            }
            if (allowedRange === "future" && viewMonth < thisMonth && viewYear <= thisYear) {
                div.classList.add("disabled");
            }
            list.push(div);
        }

        frame.innerHTML = "";
        frame.append(...list);
        year.textContent = viewYear;

        // Zero-length timeout to force animation to work.
        setTimeout(() => {
            dates.classList.add("hidden");
            month.classList.add("hidden");
            months.classList.remove("hidden");
            year.classList.remove("hidden");
        }, 0);
        return;
    }

    // Show years.
    if (!months.classList.contains("hidden")) {

        const years = q(".years", calendar);
        const year = q(".year", calendar);
        const period = q(".period", calendar);
        const frame = q(".frame", years);

        const list = [];
        const thisYear = new Date().getFullYear();

        let setYear;
        if (calendar.dataset.year) {
            setYear = parseInt(calendar.dataset.year);
        }

        const currentYear = parseInt(year.textContent);
        if (allowedRange === "past" && currentYear+1 > thisYear) {
            next.classList.add("disabled");
        } else {
            next.classList.remove("disabled");
        }
        if (allowedRange === "future" && currentYear-12 < thisYear) {
            prev.classList.add("disabled");
        } else {
            prev.classList.remove("disabled");
        }

        for (let i = 0; i < 12; i++) {
            const div = document.createElement("div");
            const viewYear = currentYear + i - 11;
            div.textContent = viewYear;
            if (setYear && viewYear === setYear) {
                div.classList.add("selected");
            }
            if (allowedRange === "past" && viewYear > thisYear) {
                div.classList.add("disabled");
            }
            if (allowedRange === "future" && viewYear < thisYear) {
                div.classList.add("disabled");
            }
            list.push(div);
        }

        frame.innerHTML = "";
        frame.append(...list);
        period.textContent = yearRange(currentYear-11, currentYear);

        // Zero-length timeout to force animation to work.
        setTimeout(() => {
            q(".view", calendar).classList.add("disabled");
            q(".months", calendar).classList.add("hidden");
            year.classList.add("hidden");
            years.classList.remove("hidden");
            period.classList.remove("hidden");
        }, 0);
    }
}

function calendarPrev(e) {

    const prev = e.currentTarget;
    const calendar = findAncestor(".calendar", prev);
    const next = q(".next", calendar);
    const allowedRange = calendar.dataset.allowedRange;

    if (calendar.dataset.animating) {
        return;
    }
    calendar.dataset.animating = true;

    const dates = q(".dates", calendar);
    const months = q(".months", calendar);
    const month = q(".month", calendar);
    const years = q(".years", calendar);
    const year = q(".year", calendar);
    const period = q(".period", calendar);
    const isDateType = calendar.dataset.type === "date";

    const now = new Date();
    const thisYear = now.getFullYear();
    const thisMonth = now.getMonth();
    const thisdate = now.getDate();

    // Dates context.
    if (isDateType && !dates.classList.contains("hidden")) {
        const monthName = month.textContent.split(" ", 2)[0];
        let yyyy = parseInt(month.textContent.slice(-4));
        let mm = monthNames.indexOf(monthName);
        mm--;
        if (mm < 0) {
            mm = 11;
            yyyy--;
        }
        if (allowedRange === "future" && (yyyy === thisYear && mm === thisMonth)) {
            prev.classList.add("disabled");
        }
        next.classList.remove("disabled");
        month.textContent = `${monthNames[mm]} ${yyyy}`;
        calendarAnimateFrames(calendar, "dates", true);
        return;
    }

    // Months context.
    if (!months.classList.contains("hidden")) {
        const prevYear = parseInt(year.textContent)-1;
        if (allowedRange === "future" && prevYear === thisYear) {
            prev.classList.add("disabled");
        }
        next.classList.remove("disabled");
        year.textContent = prevYear;
        calendarAnimateFrames(calendar, "months", true);
        return;
    }

    // Years context.
    const frame = q(".frame", years);
    const firstYear = parseInt(frame.children[0].textContent);
    if (allowedRange === "future" && firstYear-12 <= thisYear) {
        prev.classList.add("disabled");
    }
    next.classList.remove("disabled");
    period.textContent = yearRange(firstYear-12, firstYear-1);
    calendarAnimateFrames(calendar, "years", true);
}

function calendarNext(e) {

    const next = e.currentTarget;
    const calendar = findAncestor(".calendar", next);
    const prev = q(".prev", calendar);
    const allowedRange = calendar.dataset.allowedRange;

    if (calendar.dataset.animating) {
        return;
    }
    calendar.dataset.animating = true;

    const dates = q(".dates", calendar);
    const months = q(".months", calendar);
    const month = q(".month", calendar);
    const years = q(".years", calendar);
    const year = q(".year", calendar);
    const period = q(".period", calendar);
    const isDateType = calendar.dataset.type === "date";

    const now = new Date();
    const thisYear = now.getFullYear();
    const thisMonth = now.getMonth();
    const thisdate = now.getDate();

    // Dates context.
    if (isDateType && !dates.classList.contains("hidden")) {
        const monthName = month.textContent.split(" ", 2)[0];
        let yyyy = parseInt(month.textContent.slice(-4));
        let mm = monthNames.indexOf(monthName);
        mm++;
        if (mm > 11) {
            mm = 0;
            yyyy++;
        }
        if (allowedRange === "past" && (yyyy === thisYear && mm === thisMonth)) {
            next.classList.add("disabled");
        }
        prev.classList.remove("disabled");
        month.textContent = `${monthNames[mm]} ${yyyy}`;
        calendarAnimateFrames(calendar, "dates", false);
        return;
    }

    // Months context.
    if (!months.classList.contains("hidden")) {
        const nextYear = parseInt(year.textContent)+1
        if (allowedRange === "past" && nextYear === thisYear) {
            next.classList.add("disabled");
        }
        prev.classList.remove("disabled");
        year.textContent = nextYear;
        calendarAnimateFrames(calendar, "months", false);
        return;
    }

    // Years context.
    const frame = q(".frame", years);
    const last = frame.children.length-1;
    const finalYear = parseInt(frame.children[last].textContent);
    if (allowedRange === "past" && finalYear+12 >= thisYear) {
        next.classList.add("disabled");
    }
    prev.classList.remove("disabled");
    period.textContent = yearRange(finalYear+1, finalYear+12);
    calendarAnimateFrames(calendar, "years", false);
}

function calendarAnimateFrames(calendar, cls, prev) {

    let unit = q("." + cls, calendar);
    const oldFrame = q(".frame", unit);
    let newFrame = oldFrame.cloneNode(true);

    switch (cls) {
    case "dates":
        unit = q(".frames", unit);
        newFrame = calendarInitDates(calendar);
        break;
    case "months":
        calendarMonthsFrame(calendar, newFrame);
        break;
    case "years":
        calendarYearsFrame(oldFrame, newFrame, prev);
        break;
    default:
        throw "Unknown class during call to calendarAnimateFrames";
    }

    if (prev) {
        unit.classList.add("bottom");
        unit.prepend(newFrame);
    } else {
        unit.classList.add("top");
        unit.append(newFrame);
    }

    const n = getTransitionDuration(oldFrame);
    setTimeout(() => {
        unit.classList.remove("bottom");
        unit.classList.remove("top");
        removeNode(oldFrame);
        delete calendar.dataset.animating;
    }, n);
}
