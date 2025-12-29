// This file contains shared JS code for all admin views
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...


try {
    var clipboard = new ClipboardJS('.copyurl');
} catch (ignored) {
}

var toastId;

function showToast(timeout, text) {
    let notification = document.getElementById("toastnotification");
    if (typeof text !== 'undefined')
        notification.innerText = text;
    else
        notification.innerText = notification.dataset.default;
    notification.classList.add("show");

    clearTimeout(toastId);
    toastId = setTimeout(() => {
        hideToast();
    }, timeout);
}

function hideToast() {
    document.getElementById("toastnotification").classList.remove("show");
}

