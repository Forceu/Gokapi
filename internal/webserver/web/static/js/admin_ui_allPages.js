// This file contains shared JS code for all admin views
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...


var clipboard = new ClipboardJS('.btn');

function showToast(timeout, text) {
    let notification = document.getElementById("toastnotification");
    if (typeof text !== 'undefined')
        notification.innerText = text;
    else
        notification.innerText = notification.dataset.default;
    notification.classList.add("show");
    setTimeout(() => {
        notification.classList.remove("show");
    }, timeout);
}

// For some reason ClipboardJs is not working on the user PW reset modal, even when initilising again. Manually writing to clipboard
function copyToClipboard(text, timeout, toastText) {
    navigator.clipboard.writeText(text);
    showToast(timeout, toastText);
}
