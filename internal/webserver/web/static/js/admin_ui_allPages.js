// This file contains shared JS code for all admin views
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...


try {
    var clipboard = new ClipboardJS('.copyurl');
} catch (ignored) {}

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


var calendarInstance = null;

function createCalendar(element, timestamp) {
    const expiryDate = new Date(timestamp * 1000);

    calendarInstance = flatpickr(document.getElementById(element), {
        enableTime: true,
        dateFormat: 'U', // Unix timestamp
        altInput: true,
        altFormat: 'Y-m-d H:i',
        allowInput: true,
        time_24hr: true,
        defaultDate: expiryDate,
        minDate: 'today',
    });
}


function handleEditCheckboxChange(checkbox) {
    var targetElement = document.getElementById(checkbox.getAttribute("data-toggle-target"));
    var timestamp = checkbox.getAttribute("data-timestamp");

    if (checkbox.checked) {
        targetElement.classList.remove("disabled");
        targetElement.removeAttribute("disabled");
        if (timestamp != null) {
            calendarInstance._input.disabled = false;
        }
    } else {
        if (timestamp != null) {
            calendarInstance._input.disabled = true;
        }
        targetElement.classList.add("disabled");
        targetElement.setAttribute("disabled", true);
    }
}

function downloadFileWithPresign(id) {
    apiFilesListDownloadSingle(id)
        .then(data => {
            if (!data.hasOwnProperty("downloadUrl")) {
                throw new Error("Unable to get presigned key");
            }
            const a = document.createElement('a');
            a.href = data.downloadUrl;
            a.style.display = 'none';

            document.body.appendChild(a);
            a.click();
            a.remove();
        })
        .catch(error => {
            alert("Unable to download: " + error);
            console.error('Error:', error);
        });
}

function downloadFilesZipWithPresign(ids, filename) {
    apiFilesListDownloadZip(ids, filename)
        .then(data => {
            if (!data.hasOwnProperty("downloadUrl")) {
                throw new Error("Unable to get presigned key");
            }
            const a = document.createElement('a');
            a.href = data.downloadUrl;
            a.style.display = 'none';

            document.body.appendChild(a);
            a.click();
            a.remove();
        })
        .catch(error => {
            alert("Unable to download: " + error);
            console.error('Error:', error);
        });
}

/**
 * doLogout
 *
 * Called by the Logout link's onclick. Notifies the SharedWorker (if active)
 * to drop this port – and close the SSE connection if no other tabs remain –
 * then navigates to ./logout. Using an explicit message here is more precise
 * than beforeunload, which would fire on every navigation (refresh, back, etc.)
 * and could prematurely tear down the worker while other tabs are still open.
 */
function doLogout(event) {
    if (typeof sseWorkerPort !== "undefined" && sseWorkerPort !== null) {
        // "shutdown" tells the worker to close the SSE connection and notify
        // all other tabs, not just this one, since the session is now invalid
        // for everyone. Each tab's onmessage handler will redirect to ./login.
        sseWorkerPort.postMessage({ type: "shutdown" });
    }
    window.location.href = "./logout";
}
