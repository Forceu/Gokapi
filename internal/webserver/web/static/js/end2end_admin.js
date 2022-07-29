Blob.prototype.arrayBuffer ??= function() {
    return new Response(this).arrayBuffer()
}


		if (!isE2EKeySet()) {
			window.location='./e2eSetup';
		} else {
			loadWasm(function() {
				let key = localStorage.getItem("e2ekey");
				let err = GokapiE2ESetCipher(key); //TODO
				getE2EInfo();
			});
		}


function setE2eUpload() {
dropzoneObject.uploadFiles = function(files) {
    this._transformFiles(files, (transformedFiles) => {
        let transformedFile = transformedFiles[0];
        files[0].upload.chunked = true;
        files[0].isEndToEndEncrypted = true;

        let filename = files[0].upload.filename; //TODO remove filename and contenttype
        let plainTextSize = transformedFile.size;
        let bytesSent = 0;

        let encryptedSize = GokapiE2EEncryptNew(files[0].upload.uuid, plainTextSize, filename); //TODO error checking

        files[0].upload.totalChunkCount = Math.ceil(
            encryptedSize / this.options.chunkSize
        );

        files[0].sizeEncrypted = encryptedSize;
        let file = files[0];

        let bytesReadPlaintext = 0;
        let bytesSendEncrypted = 0;

        let finishedReading = false;
        let chunkIndex = 0;


        uploadChunk(file, 0, encryptedSize, plainTextSize, this.options.chunkSize);
    });
}
}

async function uploadChunk(file, chunkIndex, encryptedTotalSize, plainTextSize, chunkSize) {
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

    let err = await GokapiE2EUploadChunk(file.upload.uuid, data.byteLength, isLastChunk, new Uint8Array(data)); //TODO error checking
    data = null;
    dataBlock = null;

    if (!isLastChunk) {
        uploadChunk(file, chunkIndex + 1, encryptedTotalSize, plainTextSize, chunkSize)
    } else {
        file.status = Dropzone.SUCCESS;
        dropzoneObject.emit("success", file, 'success', null);
        dropzoneObject.emit("complete", file);
        dropzoneObject.processQueue();

        dropzoneObject.options.chunksUploaded(file, () => {
            //  dropzoneObject._finished(files, "responseText", null);
        });
    }
}
