function hash(key) {
    let hash = 0, i, chr;

    for (i = 0; i < key.length; i++) {
        chr   = key.charCodeAt(i);
        hash  = ((hash << 5) - hash) + chr;
        hash |= 0; // Convert to 32bit integer
    }

    return hash;
}

function getSession(hash, key) {
    try {
        return JSON.parse(localStorage.getItem(key + '.' + hash));
    } catch (SyntaxError) {
        return null;
    }
}

function setSession(hash, key, value) {
    localStorage.setItem(key + '.' + hash, JSON.stringify(value));
}

function clearSession(hash, key = null) {
    key ? localStorage.removeItem(key + '.' + hash) : localStorage.clear();
}

export {
    hash, getSession, setSession, clearSession
}