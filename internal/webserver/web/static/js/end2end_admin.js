Blob.prototype.arrayBuffer ??= function() {
    return new Response(this).arrayBuffer()
}
isE2EEnabled = true;


function displayError(err) {
    document.getElementById("errordiv").style.display = "block";
    document.getElementById("errormessage").innerHTML = "<b>Error: </b> " + err.toString().replace(/^Error:/gi, "");
    console.error('Caught exception', err)
}


function checkIfE2EKeyIsSet() {
    if (!isE2EKeySet()) {
        window.location = './e2eSetup';
    } else {
        loadWasm(function() {
            let key = localStorage.getItem("e2ekey");
            let err = GokapiE2ESetCipher(key);
            if (err !== null) {
                displayError(err);
                return;
            }
            getE2EInfo();
            GokapiE2EDecryptMenu();
            dropzoneObject.enable();
            document.getElementsByClassName("dz-button")[0].innerText = "Drop files, paste or click here to upload (end-to-end encrypted)";
        });
    }
}

function getE2EInfo() {
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "./e2eInfo?action=get", false);
    xhr.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                let err = GokapiE2EInfoParse(xhr.response);
                if (err !== null) {
                    displayError(err);
                    if (err.message === "cipher: message authentication failed") {
                        invalidCipherRedirectConfim();
                    }
                }
            } else {
                displayError("Trying to get E2E info: " + xhr.statusText);
            }
        }
    };

    xhr.send();
}

function invalidCipherRedirectConfim() {
    if (confirm('It appears that an invalid end-to-end encryption key has been entered. Would you like to enter the correct one?')) {
        window.location = './e2eSetup';
    }
}

function storeE2EInfo(data) {
    var xhr = new XMLHttpRequest();
    xhr.open("POST", "./e2eInfo?action=store", false);
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');

    xhr.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status != 200) {
                displayError("Trying to store E2E info: " + xhr.statusText);
            }
        }
    };
    let formData = new FormData();
    formData.append("info", data);
    xhr.send(urlencodeFormData(formData));
}

function isE2EKeySet() {
    let key = localStorage.getItem("e2ekey");
    return key !== null && key !== "";
}


function loadWasm(func) {
    const go = new Go(); // Defined in wasm_exec.js
    const WASM_URL = 'e2e.wasm?v=1';

    var wasm;

    try {
        if ('instantiateStreaming' in WebAssembly) {
            WebAssembly.instantiateStreaming(fetch(WASM_URL), go.importObject).then(function(obj) {
                wasm = obj.instance;
                go.run(wasm);
                func();
            })
        } else {
            fetch(WASM_URL).then(resp =>
                resp.arrayBuffer()
            ).then(bytes =>
                WebAssembly.instantiate(bytes, go.importObject).then(function(obj) {
                    wasm = obj.instance;
                    go.run(wasm);
                    func();
                })
            )
        }
    } catch (err) {
        displayError(err);
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


function setE2eUpload() {
    dropzoneObject.uploadFiles = function(files) {
        this._transformFiles(files, (transformedFiles) => {

            let transformedFile = transformedFiles[0];
            files[0].upload.chunked = true;
            files[0].isEndToEndEncrypted = true;

            let filename = files[0].upload.filename;
            let plainTextSize = transformedFile.size;
            let bytesSent = 0;

            let encryptedSize = GokapiE2EEncryptNew(files[0].upload.uuid, plainTextSize, filename);
            if (encryptedSize instanceof Error) {
                displayError(encryptedSize);
                return;
            }

            files[0].upload.totalChunkCount = Math.ceil(
                encryptedSize / this.options.chunkSize
            );

            files[0].sizeEncrypted = encryptedSize;
            let file = files[0];

            let bytesReadPlaintext = 0;
            let bytesSendEncrypted = 0;

            let finishedReading = false;
            let chunkIndex = 0;

            uploadChunk(file, 0, encryptedSize, plainTextSize, dropzoneObject.options.chunkSize, 0);

        });
    }
}


function decryptFileEntry(id, filename, cipher) {
    let datatable = $('#maintable').DataTable();
    const rows = datatable.rows().nodes();

    for (let i = 0; i < rows.length; i++) {
        const cell = datatable.cell(i, 0).node();
        if ("cell-name-" + id === $(cell).attr("id")) {
            datatable.cell(i, 0).data(filename);
            let urlNode = datatable.cell(i, 5).node();
            let urlLink = urlNode.querySelector("a");
            let url = urlLink.getAttribute("href");
            if (!url.includes(cipher)) {
                urlLink.setAttribute("href", url + "#" + cipher);
            }
            datatable.cell(i, 5).node(urlNode);


            let buttonNode = datatable.cell(i, 6).node();
            let button = buttonNode.querySelector("button");
            let urlButton = button.getAttribute("data-clipboard-text");
            if (!urlButton.includes(cipher)) {
                button.setAttribute("data-clipboard-text", urlButton + "#" + cipher);
            }
        datatable.cell(i, 6).node(buttonNode);
        break;
        }
    }
}


async function uploadChunk(file, chunkIndex, encryptedTotalSize, plainTextSize, chunkSize, bytesWritten) {
    let isLastChunk = false;
    let bytesReadPlaintext = chunkIndex * chunkSize;
    let readEnd = bytesReadPlaintext + chunkSize;

    if (chunkIndex === file.upload.totalChunkCount - 1) {
        isLastChunk = true;
        readEnd = plainTextSize;
    }


    let dataBlock = file.webkitSlice ?
        file.webkitSlice(bytesReadPlaintext, readEnd) :
        file.slice(bytesReadPlaintext, readEnd);

    let data = await dataBlock.arrayBuffer();

    let dataEnc = await GokapiE2EUploadChunk(file.upload.uuid, data.byteLength, isLastChunk, new Uint8Array(data));
    if (dataEnc instanceof Error) {
        displayError(data);
        return;
    }
    let err = await postChunk(file.upload.uuid, bytesWritten, encryptedTotalSize, dataEnc, file);
    if (err !== null) {
        file.accepted = false;
        dropzoneObject._errorProcessing([file], err);
        return;
    }
    bytesWritten = bytesWritten + dataEnc.byteLength;
    data = null;
    dataEnc = null;
    dataBlock = null;

    if (!isLastChunk) {
        await uploadChunk(file, chunkIndex + 1, encryptedTotalSize, plainTextSize, chunkSize, bytesWritten)
    } else {
        file.status = Dropzone.SUCCESS;
        dropzoneObject.emit("success", file, 'success', null);
        dropzoneObject.emit("complete", file);
        dropzoneObject.processQueue();

        dropzoneObject.options.chunksUploaded(file, () => {});
    }
}

async function postChunk(uuid, bytesWritten, encSize, data, file) {
    return new Promise(resolve => {
        let formData = new FormData();
        formData.append("dztotalfilesize", encSize)
        formData.append("dzchunkbyteoffset", bytesWritten)
        formData.append("dzuuid", uuid)
        formData.append("file", new Blob([data]), "encrypted.file");

        let xhr = new XMLHttpRequest();
        xhr.open("POST", "./uploadChunk");

        let progressObj = xhr.upload != null ? xhr.upload : xhr;
        progressObj.onprogress = (event) => {
            try {
                dropzoneObject.emit("uploadprogress", file, (100 * (event.loaded + bytesWritten)) / encSize, event.loaded + bytesWritten);
            } catch (e) {
                console.log(e);
            }
        }
        xhr.onreadystatechange = function() {
            if (this.readyState == 4) {
                if (this.status == 200) {
                    resolve(null);
                } else {
                    console.log(xhr.responseText);
                    resolve(xhr.responseText);
                }
            }
        };
        xhr.send(formData);
    });
}
