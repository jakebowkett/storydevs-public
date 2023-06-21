
function timeClear(e) {

	const time = findAncestor(".wdgt-time", e.currentTarget);
	const beneath = q(".beneath", time);
	const text = qAll(".inputs input", time);
	const hr = text[0];
	const min = text[1];
	const meridiem = qAll(".meridiem input", time);
	const am = meridiem[0];
	const pm = meridiem[1];

	hr.dataset.prev = "00";
	hr.value = "00";
	min.dataset.prev = "00";
	min.value = "00";
	am.checked = false;
	pm.checked = false;

	beneath.classList.add("hidden");
}

function timeMousedown(e) {

	if (targetIsInputOrMeridiem(e)) {
		return;
	}
	e.preventDefault();

	const time = findAncestor(".wdgt-time", e.target);
	const inputs = q(".inputs", time);
	const text = qAll("input", inputs);
	const hr = text[0];
	const min = text[1];
	const colon = q(".colon", inputs);

	const mid = colon.getBoundingClientRect().left;
	if (e.clientX > mid) {
		min.focus();
		return;
	}
	hr.focus();
}

function timeMouseup(e) {
	if (targetIsInputOrMeridiem(e)) {
		return;
	}
	e.preventDefault();
}

function targetIsInputOrMeridiem(e) {
	if (findAncestor(".beneath", e.target)) {
		return true;
	}
	if (findAncestor(".meridiem", e.target)) {
		return true;
	}
	if (e.target.tagName.toLowerCase() === "input") {
		return true;
	}
	return false;
}

function timeFocus(e) {
	e.target.select();
	e.target.dataset.selected = "true";
}

function timeBlur(e) {
	e.target.dataset.selected = "false";
}

function ensureLeadingZero(tf) {
	const v = tf.value;
	if (v.length < 2) {
		tf.value = "0".repeat(2 - v.length) + v;
	}
}

// Remember, preventDefault does not work with input events!
function timeInput(e) {

	const tf = e.target;
	const time = findAncestor(".wdgt-time", tf);
	const beneath = q(".beneath", time);
	const isHour = tf.classList.contains("hour");

	// Ensure leading zeroes.
	if (e.inputType.startsWith("delete")) {
		ensureLeadingZero(tf);
		tf.dataset.prev = tf.value;
		tf.dataset.selected = "false";
		return;
	}

	/*
		Restore previous value of textfield to make logic simpler
		below. This effectively negates any changes made by the
		input event which cannot be cancelled.
	*/
	tf.value = tf.dataset.prev;

	// Only deal with insertions from here onward.
	if (!e.inputType.startsWith("insert")) {
		return;
	}

	/*
		This can happen during a paste event. By restoring the
		previous value of the textfield above and returning here
		we've essentially disabled pasting.
	*/
	if (e.data === null) {
		return;
	}

	// Disallow non-numeric input.
	if (/\D/.test(e.data)) {
		return;
	}

	// Clear selected textfields.
	if (tf.dataset.selected === "true") {
		tf.value = "";
		tf.dataset.selected = "false";
	}

	// Strip leading zeroes and store value.
	let v = tf.value;
	let lastZero = v.search(/[^0]/);
	if (lastZero === -1) {
		lastZero = v.length;
	}
	v = tf.value.slice(lastZero);

	// Disallow more than two non-zero-lead numerals.
	if (v.length >= 2) {
		return;
	}

	// Allow any digit if the textfield is currently empty.
	if (v.length == 0) {
		tf.value = e.data;
		ensureLeadingZero(tf);
		tf.dataset.prev = tf.value;
		tf.dataset.selected = "false";
		if (isHour) {
			switchFromHourToMinute(tf);
		} else {
			const n = parseInt(e.data);
			if (n === 0 || n > 5) {
				q(".meridiem input", time).focus();
			}
		}
		beneath.classList.remove("hidden");
		return;
	}

	// Now the textfield contains one numeral. Restrict to appropriate values.
	const n = parseInt(v);
	if (isHour) {
		if (n > 1) {
			return;
		}
		if (parseInt(e.data) > 2) {
			return;
		}
		tf.value = v + e.data;
		tf.dataset.prev = tf.value;
		tf.dataset.selected = "false";
		switchFromHourToMinute(tf);
		beneath.classList.remove("hidden");
		return;
	}

	// If minute textfield.
	if (n > 5) {
		return;
	}
	tf.value = v + e.data;
	tf.dataset.prev = tf.value;
	tf.dataset.selected = "false";
	beneath.classList.remove("hidden");
	q(".meridiem input", time).focus();
}

function switchFromHourToMinute(tf) {
	const v = tf.value;
	if (parseInt(v) > 1) {
		const inputs = findAncestor(".inputs", tf);
		const minute = q(".min", inputs)
		minute.focus();
		return;
	}
}

function timeMeridiemFocus(e) {
	const time = findAncestor(".wdgt-time", e.target);
	const beneath = q(".beneath", time);
	beneath.classList.remove("hidden");
}
