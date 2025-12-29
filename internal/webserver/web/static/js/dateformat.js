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

function insertFormattedDate(unixTimestamp, id) {
    document.getElementById(id).innerText = formatUnixTimestamp(unixTimestamp);
}

function insertDateWithNegative(unixTimestamp, id, negative) {
    if (negative === undefined) {
        negative = "Never";
    }
    if (unixTimestamp == 0) {
        document.getElementById(id).innerText = negative;
        return;
    }
    insertFormattedDate(unixTimestamp, id);
}

function insertLastOnlineDate(unixTimestamp, id) {
    if ((Date.now() / 1000) - 120 < unixTimestamp) {
        document.getElementById(id).innerText = "Online";
        return;
    }
    insertDateWithNegative(unixTimestamp, id);
}

function insertFileRequestExpiry(unixTimestamp, id) {
    if (unixTimestamp == 0) {
        document.getElementById(id).innerText = "Never";
        return;
    }
    if ((Date.now() / 1000) > unixTimestamp) {
        document.getElementById(id).innerText = "Expired";
        return;
    }
    insertFormattedDate(unixTimestamp, id);
}
