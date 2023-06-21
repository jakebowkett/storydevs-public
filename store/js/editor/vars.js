
// Wiggle remove for character height variations
// within a line (measured in pixels). This is
// required to accomodate font-family changes
// within a single line because the characters
// may not be the same height despite having
// the same font-size property.
var editorLineThreshold = 5;

// Number of milliseconds to wait before
// allowing another double or triple click
// event to occur. Double and triple clicks
// during this wait period will be dropped.
var editorMultiClickThreshold = 500;

var edEmptyChar = "\u200b";

var editorSkipPoints = [
    " ",
    "-",
    ".",
    ",",
    ":",
    ";",
    "\\",
    '"',
    "'",
    "(",
    ")",
    "[",
    "]",
    "{",
    "}",
    "/",
    "\n",
    "\t",
];

var focusedEditor = null;
var ed = {
    anchor: {},
    focus: {},
    addFormat: [],
    removeFormat: [],
    composingStart: null,
};

// Can only nest paragraph, list, and atomic types.
var editorContainers = [
    // "blockquote",
    "figure",
];
// Can only nest some paragraph types (li, specifically).
var editorLists = [
    "ul",
    "ol",
];
// Can only nest inline types.
var editorParagraphs = [
    "p",
    "li",
    "code",
    // "pre", // code blocks
];
// Can only nest each other.
var editorInlines = [
    "b",
    "i",
    "u",
    "a",
    "mono", // inline code
    "key",  // keyboard instructions
    // "span",
];
// Cannot nest anything.
var editorAtomics = [
    "h2",
    "blockquote",
];
// Should be ignored when making selections/navigating.
var editorIgnore = [
    "div",
];