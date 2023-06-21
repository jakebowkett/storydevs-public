
class RuneIterator {
    
    constructor(node, overall) {
        overall = overall === undefined ? 0 : overall;
        let textNodes = [];
        this.lineariseTextNodes(node, textNodes);
        this.nodes = textNodes;
        this.max = node.textContent.length;
        if (overall > this.max) {
            overall = this.max;
        }
        this.seek(overall);
    }
    
    seek(overall) {
        
        const textNodes = this.nodes;
        
        // Find the text node containing overall.
        let startOffset = 0;
        let n = -1;
        for (let tn of textNodes) {
            n++;
            const len = tn.textContent.length;
            if ((startOffset + len) < overall) {
                startOffset += len;
                continue;
            }
            break;
        }

        // If offset is at the very end of a textNode we
        // push it into the start of the next node, unless
        // it was in the last node.
        let offset = overall - startOffset;
        let len = textNodes[n].textContent.length;
        if (offset === len && n !== textNodes.length-1) {
            n++;
            offset = 0;
        }
        
        // Ensure offset and overall refer to the
        // start of a rune boundary.
        if (offset !== 0) {
            const r = runeAt(textNodes[n].textContent, offset-1);
            if (r.length === 2) {
                offset--;
                overall--;
            }
        }
        
        this.n = n;
        this.offset = offset;
        this.overall = overall;
    }
    
    lineariseTextNodes(node, textNodes) {
        for (let child of Array.from(node.childNodes)) {
            if (child.nodeType === Node.ELEMENT_NODE) {
                this.lineariseTextNodes(child, textNodes);
                continue;
            }
            if (child.nodeType === Node.TEXT_NODE) {
                textNodes.push(child);
            }
        }
    }
    
    nodeRange(start, end) {
        
        let nodes = [];
        let offset = 0;
        
        for (const tn of this.nodes) {
            
            if (offset > end) {
                break;
            }
            
            let boundary = offset += tn.textContent.length;
            
            if (boundary > start) {
                nodes.push(tn);
                offset += tn.textContent.length;
                continue;
            }
        }
        
        return nodes;
    }
    
    nodesBefore(n) {
        
        let nodes = [];
        
        for (const tn of this.nodes) {
            
            if (tn === n) {
                break;
            }
            
            nodes.push(tn);
        }
        
        return nodes;
    }
    
    nodesAfter(n) {
        
        let nodes = [];
        
        for (let i = this.nodes.length-1; i >= 0; i--) {
            
            const tn = this.nodes[i];
            
            if (tn === n) {
                break;
            }
            
            nodes.unshift(tn);
        }
        
        return nodes;
    }
    
    nodesBetween(n1, n2) {
        
        let nodes = [];
        
        if (n1 === n2) {
            return nodes;
        }
        
        let seenStart = false;
        
        for (let tn of this.nodes) {
            
            if (tn === n1) {
                seenStart = true;
                continue;
            }
            
            if (tn === n2) {
                return nodes;
            }
            
            if (seenStart) {
                nodes.push(tn);
            }
        }
        
        return [];
    }
    
    current() {
        
        if (this.overall < 0 || this.overall > this.max) {
            return null;
        }
        
        let len;
        let r;
        
        if (this.overall === this.max) {
            r = "";
            len = 0;
        } else {
            r = runeAt(this.nodes[this.n].textContent, this.offset);
            len = r.length;
        }
        
        return {
            node: this.nodes[this.n],
            offset: this.offset,
            overall: this.overall,
            len: len,
            rune: r,
        };
    }
    
    // Returns metadata about the current position, then
    // moves the cursor one rune back. Returns null if it
    // is at the first rune.
    prev() {
        
        let len;
        let max = this.nodes[this.n].textContent.length;
        if (this.offset < 0 || this.offset === max) {
            len = 0;
        } else {
            runeAt(this.nodes[this.n].textContent, this.offset).length;
        }
        
        const cursor = {
            node: this.nodes[this.n],
            offset: this.offset,
            overall: this.overall,
            len: len,
        };
        
        // First node.
        if (this.n === 0) {
            
            // Preceding first rune.
            if (this.offset < 0) {
                return null;
            }
            
            this.decrement();
            return cursor;
        }
        
        // First rune of current node.
        if (this.offset === 0) {
            
            this.n--;
            
            let last = this.nodes[this.n].textContent.length;
            let idx;
            if (last === 1) {
                idx = 0;
            } else {
                idx = last-2;
            }
            
            const r = runeAt(this.nodes[this.n].textContent, idx);
            this.offset = last - r.length;
            this.overall -= r.length;
            return cursor;
        }
        
        this.decrement();
        return cursor;
    }
    
    decrement() {
        const offset = this.offset-2 < 0 ? 0 : this.offset-2;
        const r = runeAt(this.nodes[this.n].textContent, offset);
        this.offset -= r.length;
        this.overall -= r.length;
    }
    
    // Returns metadata about the current position, then
    // advances the cursor one rune forward. Returns null
    // if it has reached the last rune.
    next() {
        
        const len = this.nodes[this.n].textContent.length;
        let runeLen;
        if (this.offset === len) {
            runeLen = 0;
        } else {
            runeLen = runeAt(this.nodes[this.n].textContent, this.offset).length;
        }
        
        const cursor = {
            node: this.nodes[this.n],
            offset: this.offset,
            overall: this.overall,
            len: runeLen,
        };
        
        // Last node.
        if (this.n === this.nodes.length-1) {
            
            // Runes exceeded.
            if (this.offset === len) {
                return null;
            }
            
            this.offset += runeLen;
            this.overall += runeLen;
            return cursor;
        }
        
        // Last rune of current node.
        if (this.offset + runeLen === len) {
            this.n++;
            this.offset = 0;
            this.overall += runeLen;
            return cursor;
        }
        
        this.offset += runeLen;
        this.overall += runeLen;
        return cursor;
    }
}