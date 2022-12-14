var clipboard = new ClipboardJS('.btn');

var dropzoneObject;
var isE2EEnabled = false;

Dropzone.options.uploaddropzone = {
    paramName: "file",
    dictDefaultMessage: "Drop files, paste or click here to upload",
    createImageThumbnails: false,
    chunksUploaded: function(file, done) {
        sendChunkComplete(file, done);
    },
    init: function() {
        dropzoneObject = this;
        this.on("sending", function(file, xhr, formData) {});
        // This will be executed after the page has loaded. If e2e ist enabled, the end2end_admin.js has set isE2EEnabled to true
        if (isE2EEnabled) {
            dropzoneObject.disable();
            dropzoneObject.options.dictDefaultMessage = "Loading end-to-end encryption...";
            document.getElementsByClassName("dz-button")[0].innerText = "Loading end-to-end encryption...";
            setE2eUpload();
        }
    },
};

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
                dropzoneObject.removeFile(file);
                done();
            } else {
                file.accepted = false;
                dropzoneObject._errorProcessing([file], getErrorMessage(xhr.responseText));
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
    return "Error processing file: " + result.ErrorMessage;
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

    let buttons = '<button type="button" id="url-button-' + item.Id + '"  data-clipboard-text="' + jsonObject.Url + item.Id + '" class="copyurl btn btn-outline-light btn-sm">Copy URL</button>';
    if (item.HotlinkId !== "") {
        buttons = buttons + '<button type="button" data-clipboard-text="' + jsonObject.HotlinkUrl + item.HotlinkId + '" class="copyurl btn btn-outline-light btn-sm">Copy Hotlink</button> ';
    } else {
        if (item.RequiresClientSideDecryption === false && item.IsPasswordProtected === false) {
            buttons = buttons + '<button type="button" data-clipboard-text="' + jsonObject.GenericHotlinkUrl + item.Id + '" class="copyurl btn btn-outline-light btn-sm">Copy Hotlink</button> ';
        } else {
            buttons = buttons + '<button type="button"class="copyurl btn btn-outline-light btn-sm disabled">Copy Hotlink</button> ';
        }
    }
    buttons = buttons + "<button type=\"button\" class=\"btn btn-outline-light btn-sm\" onclick=\"window.location='./delete?id=" + item.Id + "'\">Delete</button>";

    cellButtons.innerHTML = buttons;

    cellFilename.style.backgroundColor = "green"
    cellFileSize.style.backgroundColor = "green"
    console.log(jsonObject);
    cellFileSize.setAttribute('data-order', jsonObject.FileInfo.SizeBytes);
    cellRemainingDownloads.style.backgroundColor = "green"
    cellStoredUntil.style.backgroundColor = "green"
    cellDownloadCount.style.backgroundColor = "green"
    cellUrl.style.backgroundColor = "green"
    cellButtons.style.backgroundColor = "green"
    $('#maintable').DataTable().row.add(row);
    return item.Id;
}
