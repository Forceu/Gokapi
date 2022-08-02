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
        this.on("sending", function(file, xhr, formData) {
            formData.append("allowedDownloads", document.getElementById("allowedDownloads").value);
            formData.append("expiryDays", document.getElementById("expiryDays").value);
            formData.append("password", document.getElementById("password").value);
            formData.append("isUnlimitedDownload", !document.getElementById("enableDownloadLimit").checked);
            formData.append("isUnlimitedTime", !document.getElementById("enableTimeLimit").checked);
	    if (isE2EEnabled) {
	       formData.append("isE2E", "true");
	    }
        });
        if (isE2EEnabled) {
        	setE2eUpload();
        }
    },
};

document.onpaste = function(event) {
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
	    formData.append("filename", "file.e2e");
	    formData.append("filecontenttype", "");
	    formData.append("isE2E", "true");
    } else {
	    formData.append("filesize", file.size);
	    formData.append("filename", file.name);
	    formData.append("filecontenttype", file.type);
    }

    xhr.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                Dropzone.instances[0].removeFile(file);
                let fileId = addRow(xhr.response);
                if (file.isEndToEndEncrypted === true) {
                	let err = GokapiE2EAddFile(file.upload.uuid, fileId, file.name); //TODO
                	let info = GokapiE2EInfoEncrypt(); //TODO
                	storeE2EInfo(info);
                }
                done();
            } else {
                file.accepted = false;
                Dropzone.instances[0]._errorProcessing([file], getErrorMessage(xhr.responseText));
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
    cellUrl.innerHTML = '<a  target="_blank" style="color: inherit" href="' + jsonObject.Url + item.Id + '">' + item.Id + '</a>' + lockIcon;

    let buttons = "<button type=\"button\" data-clipboard-text=\"" + jsonObject.Url + item.Id + "\" class=\"copyurl btn btn-outline-light btn-sm\">Copy URL</button> ";
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
    cellRemainingDownloads.style.backgroundColor = "green"
    cellStoredUntil.style.backgroundColor = "green"
    cellDownloadCount.style.backgroundColor = "green"
    cellUrl.style.backgroundColor = "green"
    cellButtons.style.backgroundColor = "green"
    return item.Id;
}
