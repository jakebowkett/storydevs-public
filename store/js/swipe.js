
// Minimum pixels per 100ms that must be exceeded to swipe to the next column.
var swipeSpeedMin;

// Speed in colSwipeEnd() is pixels per swipeDeltaMs.
var swipeDeltaMs = 100;

// Maximum number of seconds a swipe will take when not using momentum.
var swipeMaxSeconds = 0.3;

// Minimum number of seconds a swipe will take when not using momentum.
var swipeMinSeconds = 0.15;

var ongoingSwipe;
var colRefs;

function colSwipeStart(e) {

	// Disallow starting a new swipe while one is still in progress.
	if (ongoingSwipe) {
		return;
	}

	// We intentionally assign colSwipeEnd to cancel event here.
	const col = e.currentTarget;
	const w = col.clientWidth;
	swipeSpeedMin = w/6;
	col.addEventListener('touchmove', colSwipeMove);
	col.addEventListener('touchend', colSwipeEnd);
	col.addEventListener('touchcancel', colSwipeEnd);

	colRefs = [
		q("#search"),
		q("#browse"),
		q("#detail"),
		q("#editor"),
	];

	// Find current column position and disable all column animations.
	let n;
	for (let i = 0; i < colRefs.length; i++) {
		colRefs[i].style.transitionDuration = "0s";
		colRefs[i].style.zIndex = "1";
		if (colRefs[i] === col) {
			n = i;
		}
	}

	// Position adjacent columns.
	col.style.transform = `translateX(0px)`;
	col.style.zIndex = "2";
	if (n > 0) {
		colRefs[n-1].style.transform = `translateX(-${w}px)`;
		colRefs[n-1].style.zIndex = "2";
	}
	if (n < colRefs.length-1) {
		colRefs[n+1].style.transform = `translateX(${w}px)`;
		colRefs[n+1].style.zIndex = "2";
	}

	// Store touch start point and time.
	const touch = e.changedTouches[0];
	ongoingSwipe = {
		id: touch.identifier,
		lastAt: Date.now(),
		lastAtInterval: 0,
		lastX: touch.screenX,
		lastXDelta: 0,
		x: touch.screenX,
		y: touch.screenY,
		n: n,
		xDelta: 0,
		yDelta: 0,
	};
}

function colSwipeMove(e) {

	const col = e.currentTarget;
	const w = col.clientWidth;
	const touch = activeTouch(e);
	const now = Date.now();
	const o = ongoingSwipe;
	const n = o.n;

	// Push the info needed to calculate swipe momentum.
	o.lastXDelta = o.lastX - touch.screenX;
	o.lastX = touch.screenX;
	o.lastAtInterval = now - o.lastAt;
	o.lastAt = now;

	// Accumulate deltas.
	o.xDelta = touch.screenX - o.x;
	o.yDelta = touch.screenY - o.y;

	// If this is true we're vertically scrolling not swiping.
	if (!o.swiping && Math.abs(o.yDelta) > Math.abs(o.xDelta)) {
		removeSwipeTransforms();
		removeSwipeEvents(col);
		restoreColumnAnimations();
		ongoingSwipe = null;
		return;
	}

	// Flag current swipe as horizontal.
	o.swiping = true;

	// Translate columns.
	col.style.transform = `translateX(${o.xDelta}px)`;
	if (n > 0) {
		colRefs[n-1].style.transform = `translateX(${o.xDelta - w}px)`;
	}
	if (n < colRefs.length-1) {
		colRefs[n+1].style.transform = `translateX(${o.xDelta + w}px)`;
	}
}

function colSwipeEnd(e) {

	const col = e.currentTarget;
	const w = col.clientWidth;
	const touch = activeTouch(e);
	const now = Date.now();
	const o = ongoingSwipe;
	const n = o.n;

	removeSwipeEvents(col);

	/*
		If we're not swiping we immediately cancel/restore
		everything. The callback below restores the column's
		transitionDuration on the next frame which means if
		the user was tapping rather than swiping any tapping
		action that results in column animation would be
		affected. That is, the columns would not animation
		because their transitionDuration had not been restored.
	*/
	if (!o.swiping) {
		ongoingSwipe = null;
		removeSwipeTransforms();
		restoreColumnAnimations();
		return;
	}

	/*
		The gap between events can be 0 in rare cases which
		will cause a divide by zero bug below. Therefore we
		ensure it is at least 1.
	*/
	if (o.lastAtInterval <= 0) {
		o.lastAtInterval = 1;
	}

	let fr = swipeDeltaMs / o.lastAtInterval;
	let speed = Math.abs(o.lastXDelta) * fr;
	if (speed > w/2) {
		speed = w/2;
	}
	if (speed < swipeSpeedMin) {
		speed = swipeSpeedMin;
	}


	let momentum = 0;
	let tmp = speed;
	while (tmp >= 1) {
		momentum += tmp;
		tmp /= 3;
	}
	if (o.xDelta < 0) {
		momentum *= -1;
	}
	o.xDelta += momentum;


	/*
		Check if the column has been swiped more than 50%
		of its width across the screen in either direction.
	*/
	let base = 0;
	let layout = col.id;
	if (o.xDelta < -w/2 && n < colRefs.length-1) {
		base = -w;
		layout = colRefs[n+1].id;
	} else if (o.xDelta > w/2 && n > 0) {
		base = w;
		layout = colRefs[n-1].id;
	}

	/*
		If the swipe hasn't advanced far enough to move to the
		next column OR if it has but its speed is below the
		threshold we ease toward the target based on the fraction
		of the column width remaining.

		Otherwise we calculate the duration based on the swipe's
		momentum and continue the post-swipe animation by matching
		the speed and easing out toward the target.
	*/
	let duration;
	let tf;
	if (base === 0 || speed === swipeSpeedMin) {
		const val = getTranslateXVal(col);
		duration = swipeMaxSeconds * (Math.abs(val) / (w/2));
		if (duration < swipeMinSeconds) {
			duration = swipeMinSeconds;
		}
		tf = "ease";
	} else {
		const diff = Math.abs(base - getTranslateXVal(col));
		duration = (swipeDeltaMs/1000) * (diff / speed);
		if (duration <= 0) {
			duration = 0.1;
		}
		tf = "ease-out";
	}

	col.style.transitionDuration = `${duration}s`;
	col.style.transitionTimingFunction = tf;
	col.style.transform = `translateX(${base}px)`;
	if (n > 0) {
		colRefs[n-1].style.transitionDuration = `${duration}s`;
		colRefs[n-1].style.transitionTimingFunction = tf;
		colRefs[n-1].style.transform = `translateX(${base + -w}px)`;
	}
	if (n < colRefs.length-1) {
		colRefs[n+1].style.transitionDuration = `${duration}s`;
		colRefs[n+1].style.transitionTimingFunction = tf;
		colRefs[n+1].style.transform = `translateX(${base + w}px)`;
	}
	setLayout(layout, context.view, context.subView, {suppressSetMoving: true});
	setTimeout(() => {
		ongoingSwipe = null;
		removeSwipeTransforms();
		restoreColumnAnimations();
	}, (duration*1000) + 10);
}

function removeSwipeTransforms() {
	for (const c of colRefs) {
		c.style.transform = "";
		c.style.zIndex = "";
	}
}

function removeSwipeEvents(col) {
	col.removeEventListener("touchmove", colSwipeMove);
	col.removeEventListener("touchend", colSwipeEnd);
	col.removeEventListener("touchcancel", colSwipeEnd);
}

function restoreColumnAnimations() {
	for (const c of colRefs) {
		c.style.transitionDuration = "";
	}
}

function activeTouch(e) {
	for (let i = 0; i < e.changedTouches.length; i++) {
		if (e.changedTouches[i].identifier === ongoingSwipe.id) {
			return e.changedTouches[i];
		}
	}
}

function getTranslateXVal(elem) {
	return parseInt(
		elem.style.transform.slice(
			"translateX(".length,
			-"px)".length
		)
	);
}