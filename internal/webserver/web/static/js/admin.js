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
    chunksUploaded: function(file, done) {
        sendChunkComplete(file, done);
    },
    init: function() {
        dropzoneObject = this;
        this.on("addedfile", file => {
            addFileProgress(file);
        });
        this.on("queuecomplete", function() {
            isUploading = false;
        });
        this.on("sending", function(file, xhr, formData) {
            isUploading = true;
        });
        this.on("uploadprogress", function(file, progress, bytesSent) {
            updateProgressbar(file, progress, bytesSent);
        });

        // This will be executed after the page has loaded. If e2e ist enabled, the end2end_admin.js has set isE2EEnabled to true
        if (isE2EEnabled) {
            dropzoneObject.disable();
            dropzoneObject.options.dictDefaultMessage = "Loading end-to-end encryption...";
            document.getElementsByClassName("dz-button")[0].innerText = "Loading end-to-end encryption...";
            setE2eUpload();
        }
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

document.onpaste = function(event) {
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
            item.getAsString(function(s) {
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
    var xhr = new XMLHttpRequest();
    xhr.open("POST", "./uploadComplete", true);
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

    xhr.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                let fileId = addRow(xhr.response);
                if (file.isEndToEndEncrypted === true) {
                    try {
                        let result = GokapiE2EAddFile(file.upload.uuid, fileId, file.name);
                        if (result instanceof Error) {
                            throw result;
                        }
                        let info = GokapiE2EInfoEncrypt();
                        if (info instanceof Error) {
                            throw info;
                        }
                        storeE2EInfo(info);
                    } catch (err) {
                        file.accepted = false;
                        dropzoneObject._errorProcessing([file], err);
                        return;
                    }
                    GokapiE2EDecryptMenu();
                }
                removeFileStatus(file.upload.uuid);
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
        let eventData = JSON.parse(event.data);
        setProgressStatus(eventData.chunkid, eventData.currentstatus);
    }
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


function addRow(jsonText) {
    let jsonObject = parseData(jsonText);
    if (jsonObject.Result !== "OK") {
        alert("Failed to upload file!");
        location.reload();
        return;
    }
    let item = jsonObject.FileInfo;
    let table = document.getElementById("downloadtable");
    let row = table.insertRow(0);
    let cellFilename = row.insertCell(0);
    let cellFileSize = row.insertCell(1);
    let cellRemainingDownloads = row.insertCell(2);
    let cellStoredUntil = row.insertCell(3);
    let cellDownloadCount = row.insertCell(4);
    let cellUrl = row.insertCell(5);
    let cellButtons = row.insertCell(6);
    let lockIcon = "";

    if (item.IsPasswordProtected === true) {
        lockIcon = " &#128274;";
    }
    cellFilename.innerText = item.Name;
    cellFilename.id = "cell-name-" + item.Id;
    cellFileSize.innerText = item.Size;
    if (item.UnlimitedDownloads) {
        cellRemainingDownloads.innerText = "Unlimited";
    } else {
        cellRemainingDownloads.innerText = item.DownloadsRemaining;
    }
    if (item.UnlimitedTime) {
        cellStoredUntil.innerText = "Unlimited";
    } else {
        cellStoredUntil.innerText = item.ExpireAtString;
    }
    cellDownloadCount.innerHTML = '0';
    cellUrl.innerHTML = '<a  target="_blank" style="color: inherit" id="url-href-' + item.Id + '" href="' + jsonObject.Url + item.Id + '">' + item.Id + '</a>' + lockIcon;

    let buttons = '<button type="button" onclick="showToast()" id="url-button-' + item.Id + '"  data-clipboard-text="' + jsonObject.Url + item.Id + '" class="copyurl btn btn-outline-light btn-sm">Copy URL</button> ';
    if (item.HotlinkId !== "") {
        buttons = buttons + '<button type="button" onclick="showToast()" data-clipboard-text="' + jsonObject.HotlinkUrl + item.HotlinkId + '" class="copyurl btn btn-outline-light btn-sm">Copy Hotlink</button> ';
    } else {
        if (item.RequiresClientSideDecryption === true || item.IsPasswordProtected === true) {
            buttons = buttons + '<button type="button"class="copyurl btn btn-outline-light btn-sm disabled">Copy Hotlink</button> ';
        } else {
            buttons = buttons + '<button type="button" onclick="showToast()" data-clipboard-text="' + jsonObject.GenericHotlinkUrl + item.Id + '" class="copyurl btn btn-outline-light btn-sm">Copy Hotlink</button> ';
        }
    }
    buttons = buttons + "<button type=\"button\" class=\"btn btn-outline-light btn-sm\" onclick=\"showQrCode('" + jsonObject.Url + item.Id + "');\">QR</button> ";
    buttons = buttons + "<button type=\"button\" class=\"btn btn-outline-light btn-sm\" onclick=\"window.location='./delete?id=" + item.Id + "'\">Delete</button>";

    cellButtons.innerHTML = buttons;

    cellFilename.style.backgroundColor = "green"
    cellFileSize.style.backgroundColor = "green"
    cellFileSize.setAttribute('data-order', jsonObject.FileInfo.SizeBytes);
    cellRemainingDownloads.style.backgroundColor = "green"
    cellStoredUntil.style.backgroundColor = "green"
    cellDownloadCount.style.backgroundColor = "green"
    cellUrl.style.backgroundColor = "green"
    cellButtons.style.backgroundColor = "green"
    let datatable = $('#maintable').DataTable();

    if (rowCount == -1) {
        rowCount = datatable.rows().count();
    }
    rowCount = rowCount + 1;
    datatable.row.add(row);

    let infoEmpty = document.getElementsByClassName("dataTables_empty")[0];
    if (typeof infoEmpty !== "undefined") {
        infoEmpty.innerText = "Files stored: " + rowCount;
    } else {
        document.getElementsByClassName("dataTables_info")[0].innerText = "Files stored: " + rowCount;
    }
    return item.Id;
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
