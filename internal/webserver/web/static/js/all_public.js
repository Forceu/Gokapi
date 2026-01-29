function getUuid() {
    // Native UUID, not available in insecure environment
    if (typeof crypto !== "undefined" && crypto.randomUUID) {
        return crypto.randomUUID();
    }

    // CSPRNG-backed fallback
    if (typeof crypto !== "undefined" && crypto.getRandomValues) {
        const bytes = new Uint8Array(16);
        crypto.getRandomValues(bytes);

        // RFC 4122 compliance
        bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
        bytes[8] = (bytes[8] & 0x3f) | 0x80; // variant 10

        return [...bytes]
            .map((b, i) =>
                (i === 4 || i === 6 || i === 8 || i === 10 ? "-" : "") +
                b.toString(16).padStart(2, "0")
            )
            .join("");
    }

    // If unavailable, Math.random (not cryptographically secure)
    let uuid = "",
        i;
    for (i = 0; i < 36; i++) {
        if (i === 8 || i === 13 || i === 18 || i === 23) {
            uuid += "-";
        } else if (i === 14) {
            uuid += "4";
        } else {
            const r = Math.random() * 16 | 0;
            uuid += (i === 19 ? (r & 0x3) | 0x8 : r).toString(16);
        }
    }
    return uuid;
}


function formatUnixTimestamp(unixTimestamp) {
    const date = new Date(unixTimestamp * 1000);
    const pad = (n) => String(n).padStart(2, '0');

    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1); // months are 0-based
    const day = pad(date.getDate());
    const hours = pad(date.getHours());
    const minutes = pad(date.getMinutes());

    return `${year}-${month}-${day} ${hours}:${minutes}`;
}

function formatTimestampWithNegative(unixTimestamp, negative) {
    if (negative === undefined) {
        negative = "Never";
    }
    if (unixTimestamp == 0) {
        return negative;
    }
    return formatUnixTimestamp(unixTimestamp);
}

function insertFormattedDate(unixTimestamp, id) {
    document.getElementById(id).innerText = formatUnixTimestamp(unixTimestamp);
}

function insertDateWithNegative(unixTimestamp, id, negative) {
    document.getElementById(id).innerText = formatTimestampWithNegative(unixTimestamp, negative);
}

function insertLastOnlineDate(unixTimestamp, id) {
    if ((Date.now() / 1000) - 120 < unixTimestamp) {
        document.getElementById(id).innerText = "Online";
        return;
    }
    insertDateWithNegative(unixTimestamp, id);
}

function formatFileRequestExpiry(unixTimestamp) {
    if (unixTimestamp == 0) {
        return "Never";
    }
    if ((Date.now() / 1000) > unixTimestamp) {
        return "Expired";
    }
    return formatUnixTimestamp(unixTimestamp);
}

function insertFileRequestExpiry(unixTimestamp, id) {
    document.getElementById(id).innerText = formatFileRequestExpiry(unixTimestamp);

}

function getReadableSize(bytes) {
    if (!bytes || bytes == 0) return "0 B";
    const units = ["B", "kB", "MB", "GB", "TB"];
    let i = 0;
    while (bytes >= 1024 && i < units.length - 1) {
        bytes /= 1024;
        i++;
    }
    return `${bytes.toFixed(1)} ${units[i]}`;
}


function getReadableSizeInUnit(bytes, unit) {
    if (!bytes || bytes == 0) return "0 B";
    const units = ["B", "kB", "MB", "GB", "TB"];
    let i = 0;
    while (units[i]!=unit && i < units.length - 1) {
        bytes /= 1024;
        i++;
    }
    return `${bytes.toFixed(1)}`;
}


function insertReadableSize(bytes, multiplier, id) {
    document.getElementById(id).innerText = getReadableSize(bytes * multiplier);
}

function insertReadableSizeForcedUnit(bytes, id, unit) {
    document.getElementById(id).innerText = getReadableSizeInUnit(bytes, unit);
}

function insertReadableSizeTwoOutputs(bytes, id, idUnit) {
    let calcNumber;
    let unit;
    if (bytes < 1024) {
        calcNumber = bytes;
        unit = "B";
    } else {
        let result = getReadableSize(bytes);
        calcNumber = result.slice(0, -3);
        unit = result.slice(-2);
    }

    document.getElementById(id).innerText = calcNumber;
    document.getElementById(idUnit).innerText = unit;
}

/**
*
*  Base64 encode / decode
*  http://www.webtoolkit.info/
*
**/
var Base64={_keyStr:"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=",encode:function(r){var t,e,o,a,h,n,c,d="",C=0;for(r=Base64._utf8_encode(r);C<r.length;)a=(t=r.charCodeAt(C++))>>2,h=(3&t)<<4|(e=r.charCodeAt(C++))>>4,n=(15&e)<<2|(o=r.charCodeAt(C++))>>6,c=63&o,isNaN(e)?n=c=64:isNaN(o)&&(c=64),d=d+this._keyStr.charAt(a)+this._keyStr.charAt(h)+this._keyStr.charAt(n)+this._keyStr.charAt(c);return d},decode:function(r){var t,e,o,a,h,n,c="",d=0;for(r=r.replace(/[^A-Za-z0-9\+\/\=]/g,"");d<r.length;)t=this._keyStr.indexOf(r.charAt(d++))<<2|(a=this._keyStr.indexOf(r.charAt(d++)))>>4,e=(15&a)<<4|(h=this._keyStr.indexOf(r.charAt(d++)))>>2,o=(3&h)<<6|(n=this._keyStr.indexOf(r.charAt(d++))),c+=String.fromCharCode(t),64!=h&&(c+=String.fromCharCode(e)),64!=n&&(c+=String.fromCharCode(o));return c=Base64._utf8_decode(c)},_utf8_encode:function(r){r=r.replace(/\r\n/g,"\n");for(var t="",e=0;e<r.length;e++){var o=r.charCodeAt(e);o<128?t+=String.fromCharCode(o):o>127&&o<2048?(t+=String.fromCharCode(o>>6|192),t+=String.fromCharCode(63&o|128)):(t+=String.fromCharCode(o>>12|224),t+=String.fromCharCode(o>>6&63|128),t+=String.fromCharCode(63&o|128))}return t},_utf8_decode:function(r){for(var t="",e=0,o=c1=c2=0;e<r.length;)(o=r.charCodeAt(e))<128?(t+=String.fromCharCode(o),e++):o>191&&o<224?(c2=r.charCodeAt(e+1),t+=String.fromCharCode((31&o)<<6|63&c2),e+=2):(c2=r.charCodeAt(e+1),c3=r.charCodeAt(e+2),t+=String.fromCharCode((15&o)<<12|(63&c2)<<6|63&c3),e+=3);return t}};
