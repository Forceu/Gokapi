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
            saveUploadDefaults();
            addFileProgress(file);
        });
        this.on("queuecomplete", function() {
            isUploading = false;
        });
        this.on("sending", function(file, xhr, formData) {
            isUploading = true;
        });

        // Error handling for chunk upload, especially returning 413 error code (invalid nginx configuration)
        this.on("error", function(file, errorMessage, xhr) {
            if (xhr && xhr.status === 413) {
                showError(file, "File too large to upload. If you are using a reverse proxy, make sure that the allowed body size is at least 70MB.");
            } else {
                showError(file, "Error: " + errorMessage);
            }
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
    document.getElementById(`us-progressbar-${chunkId}`).style.width = rounded + "%";

    let uploadSpeed = Math.round(megabytePerSecond * 10) / 10;
    if (!Number.isNaN(uploadSpeed))
        document.getElementById(`us-progress-info-${chunkId}`).innerText = rounded + "% - " + uploadSpeed + "MB/s";
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

function setUploadDefaults() {
    let defaultDownloads = getLocalStorageWithDefault("defaultDownloads", 1);
    let defaultExpiry = getLocalStorageWithDefault("defaultExpiry", 14);
    let defaultPassword = getLocalStorageWithDefault("defaultPassword", "");
    let defaultUnlimitedDownloads = getLocalStorageWithDefault("defaultUnlimitedDownloads", false) === "true";
    let defaultUnlimitedTime = getLocalStorageWithDefault("defaultUnlimitedTime", false) === "true";

    document.getElementById("allowedDownloads").value = defaultDownloads;
    document.getElementById("expiryDays").value = defaultExpiry;
    document.getElementById("password").value = defaultPassword;
    document.getElementById("enableDownloadLimit").checked = !defaultUnlimitedDownloads;
    document.getElementById("enableTimeLimit").checked = !defaultUnlimitedTime;

    if (defaultPassword === "") {
        document.getElementById("enablePassword").checked = false;
        document.getElementById("password").disabled = true;
    } else {
        document.getElementById("enablePassword").checked = true;
        document.getElementById("password").disabled = false;
    }

    if (defaultUnlimitedDownloads) {
        document.getElementById("allowedDownloads").disabled = true;
    }
    if (defaultUnlimitedTime) {
        document.getElementById("expiryDays").disabled = true;
    }

}

function saveUploadDefaults() {
    localStorage.setItem("defaultDownloads", document.getElementById("allowedDownloads").value);
    localStorage.setItem("defaultExpiry", document.getElementById("expiryDays").value);
    localStorage.setItem("defaultPassword", document.getElementById("password").value);
    localStorage.setItem("defaultUnlimitedDownloads", !document.getElementById("enableDownloadLimit").checked);
    localStorage.setItem("defaultUnlimitedTime", !document.getElementById("enableTimeLimit").checked);
}

function getLocalStorageWithDefault(key, default_value) {
    var value = localStorage.getItem(key);
    if (value === null) {
        return default_value;
    }
    return value;
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
                done();
            	let progressText = document.getElementById(`us-progress-info-${file.upload.uuid}`);
            	if (progressText != null)
    			progressText.innerText = "In Queue...";
        } else {
                dropzoneUploadError(file, getErrorMessage(xhr.responseText));
            }
            } 
    };
    xhr.send(urlencodeFormData(formData));
}

function dropzoneUploadError(file, errormessage) {
	file.accepted = false;
        dropzoneObject._errorProcessing([file], errormessage);
        showError(file, errormessage);
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

function dropzoneGetFile(uid) {
    for (let i = 0; i < dropzoneObject.files.length; i++) {
        const currentFile = dropzoneObject.files[i];
        if (currentFile.upload.uuid === uid) {
            return currentFile; 
        }
    }
    return null;
}

function requestFileInfo(fileId, uid) {

//TODO add to API documentation	
    let apiUrl = './api/files/list/'+fileId;
    const requestOptions = {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,

        },
    };

    // Send the request
    fetch(apiUrl, requestOptions)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Request failed with status: ${response.status}`);
            }
            return response.json();
            
        })
        .then(data => {
            addRow(data);
             let file = dropzoneGetFile(uid);
            if (file == null) {
            	return;
            }
            if (file.isEndToEndEncrypted === true) {
                    try {
                        let result = GokapiE2EAddFile(uid, fileId, file.name);
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
                removeFileStatus(uid);
        })
        .catch(error => {
        let file = dropzoneGetFile(uid);
            if (file != null) {
            	dropzoneUploadError(file, error);
            }
            console.error('Error:', error);
        });
}



function parseProgressStatus(eventData) {
    let container = document.getElementById(`us-container-${eventData.chunk_id}`);
    if (container == null) {
        return;
    }
    container.setAttribute('data-complete', 'true');
    let text;
    switch (eventData.upload_status) {
        case 0:
            text = "Processing file...";
            break;
        case 1:
            text = "Uploading file...";
            break;
        case 2:
            text = "Finalising...";
            requestFileInfo(eventData.file_id, eventData.chunk_id);
            break;
        case 3:
            text = "Error";
            let file = dropzoneGetFile(eventData.chunk_id);
            if (eventData.error_message == "")
            	eventData.error_message = "Server Error";
            if (file != null) {
            	dropzoneUploadError(file, eventData.error_message);
            }
            return;
       default:
            text = "Unknown status";
            break;
    }
    document.getElementById(`us-progress-info-${eventData.chunk_id}`).innerText = text;
}

function showError(file, message) {
    let chunkId = file.upload.uuid;
    document.getElementById(`us-progressbar-${chunkId}`).style.width = "100%";
    document.getElementById(`us-progressbar-${chunkId}`).style.backgroundColor = "red";
    document.getElementById(`us-progress-info-${chunkId}`).innerText = message;
    document.getElementById(`us-progress-info-${chunkId}`).classList.add('uploaderror');
}


function editFile() {
    const button = document.getElementById('mb_save');
    button.disabled = true;
    let apiUrl = './api/files/modify';

    let allowedDownloads = document.getElementById('mi_edit_down').value;
    let expiryTimestamp = document.getElementById('mi_edit_expiry').value;
    let password = document.getElementById('mi_edit_pw').value;
    let originalPassword = (password === '(unchanged)');

    if (!document.getElementById('mc_download').checked) {
        allowedDownloads = 0;
    }
    if (!document.getElementById('mc_expiry').checked) {
        expiryTimestamp = 0;
    }
    if (!document.getElementById('mc_password').checked) {
        originalPassword = false;
        password = "";
    }

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'id': button.getAttribute('data-fileid'),
            'apikey': systemKey,
            'allowedDownloads': allowedDownloads,
            'expiryTimestamp': expiryTimestamp,
            'password': password,
            'originalPassword': originalPassword

        },
    };

    // Send the request
    fetch(apiUrl, requestOptions)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Request failed with status: ${response.status}`);
            }
        })
        .then(data => {
            location.reload();
        })
        .catch(error => {
            alert("Unable to edit file: " + error);
            console.error('Error:', error);
            button.disabled = false;
        });
}

var calendarInstance = null;

function createCalendar(timestamp) {
    // Convert Unix timestamp to JavaScript Date object
    const expiryDate = new Date(timestamp * 1000);

    calendarInstance = flatpickr('#mi_edit_expiry', {
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

function showEditModal(filename, id, downloads, expiry, password, unlimitedown, unlimitedtime) {
    document.getElementById("m_filenamelabel").innerHTML = filename;
    document.getElementById("mc_expiry").setAttribute("data-timestamp", expiry);
    document.getElementById("mb_save").setAttribute('data-fileid', id);
    createCalendar(expiry);

    if (unlimitedown) {
        document.getElementById("mi_edit_down").value = "1";
        document.getElementById("mi_edit_down").disabled = true;
        document.getElementById("mc_download").checked = false;
    } else {
        document.getElementById("mi_edit_down").value = downloads;
        document.getElementById("mi_edit_down").disabled = false;
        document.getElementById("mc_download").checked = true;
    }

    if (unlimitedtime) {
        document.getElementById("mi_edit_expiry").value = add14DaysIfBeforeCurrentTime(expiry);
        document.getElementById("mi_edit_expiry").disabled = true;
        document.getElementById("mc_expiry").checked = false;
        calendarInstance._input.disabled = true;
    } else {
        document.getElementById("mi_edit_expiry").value = expiry;
        document.getElementById("mi_edit_expiry").disabled = false;
        document.getElementById("mc_expiry").checked = true;
        calendarInstance._input.disabled = false;
    }

    if (password) {
        document.getElementById("mi_edit_pw").value = "(unchanged)";
        document.getElementById("mi_edit_pw").disabled = false;
        document.getElementById("mc_password").checked = true;
    } else {
        document.getElementById("mi_edit_pw").value = "";
        document.getElementById("mi_edit_pw").disabled = true;
        document.getElementById("mc_password").checked = false;
    }
    new bootstrap.Modal('#modaledit', {}).show();
}

function selectTextForPw(input) {
    if (input.value === "(unchanged)") {
        input.setSelectionRange(0, input.value.length);
    }
}

function add14DaysIfBeforeCurrentTime(unixTimestamp) {
    let currentTime = Date.now();
    let timestampInMilliseconds = unixTimestamp * 1000;
    if (timestampInMilliseconds < currentTime) {
        let newTimestamp = currentTime + (14 * 24 * 60 * 60 * 1000);
        return Math.floor(newTimestamp / 1000);
    } else {
        return unixTimestamp;
    }
}

function changeApiPermission(apiKey, permission, buttonId) {

    var indicator = document.getElementById(buttonId);
    if (indicator.classList.contains("apiperm-processing")) {
        return;
    }
    var wasGranted = indicator.classList.contains("apiperm-granted");
    indicator.classList.add("apiperm-processing");
    indicator.classList.remove("apiperm-granted");
    indicator.classList.remove("apiperm-notgranted");

    var apiUrl = './api/auth/modify';
    var modifier = "GRANT";
    if (wasGranted) {
        modifier = "REVOKE";
    }

    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'apiKeyToModify': apiKey,
            'permission': permission,
            'permissionModifier': modifier

        },
    };

    fetch(apiUrl, requestOptions)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Request failed with status: ${response.status}`);
            }
        })
        .then(data => {
            if (wasGranted) {
                indicator.classList.add("apiperm-notgranted");
            } else {
                indicator.classList.add("apiperm-granted");
            }
            indicator.classList.remove("apiperm-processing");
        })
        .catch(error => {
            if (wasGranted) {
                indicator.classList.add("apiperm-granted");
            } else {
                indicator.classList.add("apiperm-notgranted");
            }
            indicator.classList.remove("apiperm-processing");
            alert("Unable to set permission: " + error);
            console.error('Error:', error);
        });
}

function deleteApiKey(apiKey) {

    document.getElementById("delete-" + apiKey).disabled = true;
    var apiUrl = './api/auth/delete';
    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'apiKeyToModify': apiKey,
        },
    };

    fetch(apiUrl, requestOptions)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Request failed with status: ${response.status}`);
            }
        })
        .then(data => {
            document.getElementById("row-" + apiKey).remove();
        })
        .catch(error => {
            alert("Unable to delete API key: " + error);
            console.error('Error:', error);
        });
}



function newApiKey() {

    document.getElementById("button-newapi").disabled = true;
    var apiUrl = './api/auth/create';
    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'basicPermissions': 'true'
        },
    };

    fetch(apiUrl, requestOptions)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Request failed with status: ${response.status}`);
            }
        })
        .then(data => {
            location.reload();
        })
        .catch(error => {
            alert("Unable to create API key: " + error);
            console.error('Error:', error);
        });
}


function deleteFile(id) {

    document.getElementById("button-delete-" + id).disabled = true;
    var apiUrl = './api/files/delete';
    const requestOptions = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'apikey': systemKey,
            'id': id
        },
    };

    fetch(apiUrl, requestOptions)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Request failed with status: ${response.status}`);
            }
        })
        .then(data => {
            location.reload();
        })
        .catch(error => {
            alert("Unable to delete file: " + error);
            console.error('Error:', error);
        });
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

function parseSseData(data) {
    let eventData;
    try {
        eventData = JSON.parse(data);
    } catch (e) {
        console.error("Failed to parse event data:", e);
        return;
    }
    switch (eventData.event) {
        case "download":
            setNewDownloadCount(eventData.file_id, eventData.download_count, eventData.downloads_remaining);
            return;
        case "uploadStatus":
            parseProgressStatus(eventData);
            return;
        default:
            console.error("Unknown event", eventData);
    }
}

function setNewDownloadCount(id, downloadCount, downloadsRemaining) {
    let downloadCell = document.getElementById("cell-downloads-" + id);
    if (downloadCell != null) {
        downloadCell.innerHTML = downloadCount;
        downloadCell.classList.add("updatedDownloadCount");
        setTimeout(() => downloadCell.classList.remove("updatedDownloadCount"), 500);
    }
    if (downloadsRemaining != -1) {
        let downloadsRemainingCell = document.getElementById("cell-downloadsRemaining-" + id);
        if (downloadsRemainingCell != null) {
            downloadsRemainingCell.innerHTML = downloadsRemaining;
            downloadsRemainingCell.classList.add("updatedDownloadCount");
            setTimeout(() => downloadsRemainingCell.classList.remove("updatedDownloadCount"), 500);
        }
    }
}


function registerChangeHandler() {
    const source = new EventSource("./uploadStatus")
    source.onmessage = (event) => {
        parseSseData(event.data);
    }
    source.onerror = (error) => {

        // Check for net::ERR_HTTP2_PROTOCOL_ERROR 200 (OK) and ignore it
        if (error.target.readyState !== EventSource.CLOSED) {
            source.close();
        }
        console.log("Reconnecting to SSE...");
        // Attempt to reconnect after a delay
        setTimeout(registerChangeHandler, 5000);
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


function addRow(item) {
    let table = document.getElementById("downloadtable");
    let row = table.insertRow(0);
    row.id = "row-" + item.Id;
    let cellFilename = row.insertCell(0);
    let cellFileSize = row.insertCell(1);
    let cellRemainingDownloads = row.insertCell(2);
    let cellStoredUntil = row.insertCell(3);
    let cellDownloadCount = row.insertCell(4);
    let cellUrl = row.insertCell(5);
    let cellButtons = row.insertCell(6);
    let lockIcon = "";

    if (item.IsPasswordProtected === true) {
        lockIcon = '  <i  title="Password protected" class="bi bi-key"></i>';
    }
    cellFilename.innerText = item.Name;
    cellFilename.id = "cell-name-" + item.Id;
    cellDownloadCount.id = "cell-downloads-" + item.Id;
    cellFileSize.innerText = item.Size;
    if (item.UnlimitedDownloads) {
        cellRemainingDownloads.innerText = "Unlimited";
    } else {
        cellRemainingDownloads.innerText = item.DownloadsRemaining;
        cellRemainingDownloads.id = "cell-downloadsRemaining-" + item.Id;
    }
    if (item.UnlimitedTime) {
        cellStoredUntil.innerText = "Unlimited";
    } else {
        cellStoredUntil.innerText = item.ExpireAtString;
    }
    cellDownloadCount.innerHTML = '0';
    cellUrl.innerHTML = '<a  target="_blank" style="color: inherit" id="url-href-' + item.Id + '" href="' + item.UrlDownload + '">' + item.Id + '</a>' + lockIcon;

    let buttons = '<button type="button" onclick="showToast()" id="url-button-' + item.Id + '"  data-clipboard-text="' + item.UrlDownload + '" class="copyurl btn btn-outline-light btn-sm"><i class="bi bi-copy"></i> URL</button> ';
    if (item.UrlHotlink === "") {
        buttons = buttons + '<button type="button"class="copyurl btn btn-outline-light btn-sm disabled"><i class="bi bi-copy"></i> Hotlink</button> ';
    } else {
        buttons = buttons + '<button type="button" onclick="showToast()" data-clipboard-text="' + item.UrlHotlink + '" class="copyurl btn btn-outline-light btn-sm"><i class="bi bi-copy"></i> Hotlink</button> ';
    }
    buttons = buttons + '<button type="button" id="qrcode-' + item.Id + '" title="QR Code" class="btn btn-outline-light btn-sm" onclick="showQrCode(\'' + item.UrlDownload + '\');"><i class="bi bi-qr-code"></i></button> ';
    buttons = buttons + '<button type="button" title="Edit" class="btn btn-outline-light btn-sm" onclick="showEditModal(\'' + item.Name + '\',\'' + item.Id + '\', ' + item.DownloadsRemaining + ', ' + item.ExpireAt + ', ' + item.IsPasswordProtected + ', ' + item.UnlimitedDownloads + ', ' + item.UnlimitedTime + ');"><i class="bi bi-pencil"></i></button> ';
    buttons = buttons + '<button type="button" id="button-delete-' + item.Id + '" title="Delete" class="btn btn-outline-danger btn-sm" onclick="deleteFile(\'' + item.Id + '\')"><i class="bi bi-trash3"></i></button>';

    cellButtons.innerHTML = buttons;

    cellFilename.classList.add('newItem');
    cellFileSize.classList.add('newItem');
    cellRemainingDownloads.classList.add('newItem');
    cellRemainingDownloads.style.backgroundColor = "green";
    cellStoredUntil.classList.add('newItem');
    cellDownloadCount.classList.add('newItem');
    cellDownloadCount.style.backgroundColor = "green";
    cellUrl.classList.add('newItem');
    cellButtons.classList.add('newItem');
    cellFileSize.setAttribute('data-order', item.SizeBytes);
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
