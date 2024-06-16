var clipboard = new ClipboardJS('.btn');

var dropzoneObject;
var isE2EEnabled = false;

var isUploading = false;

var rowCount = -1;

window.addEventListener('beforeunload', (event) => {
    if (isUploading) {
        event.returnValue = 'Upload is still in progress. Do you want to close this page?';
    }
});

Dropzone.options.uploaddropzone = {
    paramName: "file",
    dictDefaultMessage: "Drop files, paste or click here to upload",
    createImageThumbnails: false,
    chunksUploaded: function (file, done) {
        sendChunkComplete(file, done);
    },
    init: function () {
        dropzoneObject = this;
        this.on("addedfile", file => {
            addFileProgress(file);
        });
        this.on("queuecomplete", function () {
            isUploading = false;
        });
        this.on("sending", function (file, xhr, formData) {
            isUploading = true;
        });

        // Error handling for chunk upload, especially returning 413 error code (invalid nginx configuration)
        this.on("error", function (file, errorMessage, xhr) {
            if (xhr && xhr.status === 413) {
                showError(file, "File too large to upload. If you are using a reverse proxy, make sure that the allowed body size is at least 50MB.");
            } else {
                showError(file, "Server responded with code " + xhr.status);
            }
        });

        this.on("uploadprogress", function (file, progress, bytesSent) {
            updateProgressbar(file, progress, bytesSent);
        });
    },
};



function updateProgressbar(file, progress, bytesSent) {
    let chunkId = file.upload.uuid;
    let container = document.getElementById(`us-container-${chunkId}`);
    if (container == null || container.getAttribute('data-complete') === "true") {
        return;
    }
    let rounded = Math.round(progress);
    if (rounded < 0) {
        rounded = 0;
    }
    if (rounded > 100) {
        rounded = 100;
    }
    let millisSinceUpload = Date.now() - container.getAttribute('data-starttime');
    let megabytePerSecond = bytesSent / (millisSinceUpload / 1000) / 1024 / 1024;
    let uploadSpeed = Math.round(megabytePerSecond * 10) / 10;
    document.getElementById(`us-progressbar-${chunkId}`).style.width = rounded + "%";
    document.getElementById(`us-progress-info-${chunkId}`).innerText = rounded + "% - " + uploadSpeed + "MB/s";
}

function setProgressStatus(chunkId, progressCode) {
    let container = document.getElementById(`us-container-${chunkId}`);
    if (container == null) {
        return;
    }
    container.setAttribute('data-complete', 'true');
    let text;
    switch (progressCode) {
        case 0:
            text = "Processing file...";
            break;
        case 1:
            text = "Uploading file...";
            break;
    }
    document.getElementById(`us-progress-info-${chunkId}`).innerText = text;
}

function addFileProgress(file) {
    addFileStatus(file.upload.uuid, file.upload.filename);
}

document.onpaste = function (event) {
    if (dropzoneObject.disabled) {
        return;
    }
    var items = (event.clipboardData || event.originalEvent.clipboardData).items;
    for (index in items) {
        var item = items[index];
        if (item.kind === 'file') {
            dropzoneObject.addFile(item.getAsFile());
        }
        if (item.kind === 'string') {
            item.getAsString(function (s) {
                // If a picture was copied from a website, the origin information might be submitted, which is filtered with this regex out
                const pattern = /<img *.+>/gi;
                if (pattern.test(s) === false) {
                    let blob = new Blob([s], {
                        type: 'text/plain'
                    });
                    let file = new File([blob], "Pasted Text.txt", {
                        type: "text/plain",
                        lastModified: new Date(0)
                    });
                    dropzoneObject.addFile(file);
                }
            });
        }
    }
}

function urlencodeFormData(fd) {
    let s = '';

    function encode(s) {
        return encodeURIComponent(s).replace(/%20/g, '+');
    }
    for (var pair of fd.entries()) {
        if (typeof pair[1] == 'string') {
            s += (s ? '&' : '') + encode(pair[0]) + '=' + encode(pair[1]);
        }
    }
    return s;
}

function sendChunkComplete(file, done) {
    const token = document.querySelector("#uploaddropzone").attributes.getNamedItem("data-token").value;
    var xhr = new XMLHttpRequest();
    xhr.open("POST", "./guestUploadComplete?token=" + token, true);
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');

    let formData = new FormData();
    formData.append("allowedDownloads", document.getElementById("allowedDownloads").value);
    formData.append("expiryDays", document.getElementById("expiryDays").value);
    formData.append("password", document.getElementById("password").value);
    formData.append("isUnlimitedDownload", !document.getElementById("enableDownloadLimit").checked);
    formData.append("isUnlimitedTime", !document.getElementById("enableTimeLimit").checked);
    formData.append("chunkid", file.upload.uuid);

    if (file.isEndToEndEncrypted === true) {
        formData.append("filesize", file.sizeEncrypted);
        formData.append("filename", "Encrypted File");
        formData.append("filecontenttype", "");
        formData.append("isE2E", "true");
        formData.append("realSize", file.size);
    } else {
        formData.append("filesize", file.size);
        formData.append("filename", file.name);
        formData.append("filecontenttype", file.type);
    }

    xhr.onreadystatechange = function () {
        if (this.readyState == 4) {
            if (this.status == 200) {
                removeFileStatus(file.upload.uuid);
                showUploadResult(xhr.response)
                done();
            } else {
                file.accepted = false;
                let errorMessage = getErrorMessage(xhr.responseText)
                dropzoneObject._errorProcessing([file], errorMessage);
                showError(file, errorMessage);
            }
        }
    };
    xhr.send(urlencodeFormData(formData));
}

function showUploadResult(response) {
    const result = JSON.parse(response)

    document.querySelector("#card-title").textContent = "Guest Upload Succeeded"

    document.querySelector("#upload-interface").style.display = "none";
    document.querySelector("#result-interface").style.display = "inline";

    document.querySelector("#result-name").textContent = result.FileInfo.Name;

    const link = document.createElement("a");
    link.setAttribute("href", result.Url + result.FileInfo.Id);
    link.textContent = result.Url + result.FileInfo.Id;
    document.querySelector("#result-link").appendChild(link);
    document.querySelector("#qr-button").addEventListener("click", () => showQrCode(result.Url + result.FileInfo.Id))
    document.querySelector("#url-button").setAttribute("data-clipboard-text", result.Url + result.FileInfo.Id)
}

function getErrorMessage(response) {
    let result;
    try {
        result = JSON.parse(response);
    } catch (e) {
        return "Unknown error: Server could not process file";
    }
    return "Error: " + result.ErrorMessage;
}

function showError(file, message) {
    let chunkId = file.upload.uuid;
    document.getElementById(`us-progressbar-${chunkId}`).style.width = "100%";
    document.getElementById(`us-progressbar-${chunkId}`).style.backgroundColor = "red";
    document.getElementById(`us-progress-info-${chunkId}`).innerText = message;
    document.getElementById(`us-progress-info-${chunkId}`).classList.add('uploaderror');
}

function checkBoxChanged(checkBox, correspondingInput) {
    let disable = !checkBox.checked;

    if (disable) {
        document.getElementById(correspondingInput).setAttribute("disabled", "");
    } else {
        document.getElementById(correspondingInput).removeAttribute("disabled");
    }
    if (correspondingInput === "password" && disable) {
        document.getElementById("password").value = "";
    }
}

function parseData(data) {
    if (!data) return {
        "Result": "error"
    };
    if (typeof data === 'object') return data;
    if (typeof data === 'string') return JSON.parse(data);

    return {
        "Result": "error"
    };
}

function registerChangeHandler() {
    const source = new EventSource("./uploadStatus?stream=changes")
    source.onmessage = (event) => {
        try {
            let eventData = JSON.parse(event.data);
            setProgressStatus(eventData.chunkid, eventData.currentstatus);
        } catch (e) {
            console.error("Failed to parse event data:", e);
        }
    }
    source.onerror = (error) => {

        // Check for net::ERR_HTTP2_PROTOCOL_ERROR 200 (OK) and ignore it
        if (error.target.readyState !== EventSource.CLOSED) {
            source.close();
        }


        console.log("Reconnecting to SSE...");
        // Attempt to reconnect after a delay
        setTimeout(registerChangeHandler, 1000);
    };
}

var statusItemCount = 0;


function addFileStatus(chunkId, filename) {
    const container = document.createElement('div');
    container.setAttribute('id', `us-container-${chunkId}`);
    container.classList.add('us-container');

    // create filename div
    const filenameDiv = document.createElement('div');
    filenameDiv.classList.add('filename');
    filenameDiv.textContent = filename;
    container.appendChild(filenameDiv);

    // create progress bar container div
    const progressContainerDiv = document.createElement('div');
    progressContainerDiv.classList.add('upload-progress-container');
    progressContainerDiv.setAttribute('id', `us-progress-container-${chunkId}`);

    // create progress bar div
    const progressBarDiv = document.createElement('div');
    progressBarDiv.classList.add('upload-progress-bar');

    // create progress bar progress div
    const progressBarProgressDiv = document.createElement('div');
    progressBarProgressDiv.setAttribute('id', `us-progressbar-${chunkId}`);
    progressBarProgressDiv.classList.add('upload-progress-bar-progress');
    progressBarProgressDiv.style.width = '0%';
    progressBarDiv.appendChild(progressBarProgressDiv);

    // create progress info div
    const progressInfoDiv = document.createElement('div');
    progressInfoDiv.setAttribute('id', `us-progress-info-${chunkId}`);
    progressInfoDiv.classList.add('upload-progress-info');
    progressInfoDiv.textContent = '0%';

    // append progress bar and progress info to progress bar container
    progressContainerDiv.appendChild(progressBarDiv);
    progressContainerDiv.appendChild(progressInfoDiv);

    // append progress bar container to container
    container.appendChild(progressContainerDiv);

    container.setAttribute('data-starttime', Date.now());
    container.setAttribute('data-complete', "false");

    const uploadstatusContainer = document.getElementById("uploadstatus");
    uploadstatusContainer.appendChild(container);
    uploadstatusContainer.style.visibility = "visible";
    statusItemCount++;
}

function removeFileStatus(chunkId) {
    const container = document.getElementById(`us-container-${chunkId}`);
    if (container == null) {
        return;
    }
    container.remove();
    statusItemCount--;
    if (statusItemCount < 1) {
        document.getElementById("uploadstatus").style.visibility = "hidden";
    }
}


function hideQrCode() {
    document.getElementById("qroverlay").style.display = "none";
    document.getElementById("qrcode").innerHTML = "";
}


function showQrCode(url) {
    const overlay = document.getElementById("qroverlay");
    overlay.style.display = "block";
    new QRCode(document.getElementById("qrcode"), {
        text: url,
        width: 200,
        height: 200,
        colorDark: "#000000",
        colorLight: "#ffffff",
        correctLevel: QRCode.CorrectLevel.H
    });
    overlay.addEventListener("click", hideQrCode);
}

function showToast() {
    let notification = document.getElementById("toastnotification");
    notification.classList.add("show");
    setTimeout(() => {
        notification.classList.remove("show");
    }, 1000);
}
