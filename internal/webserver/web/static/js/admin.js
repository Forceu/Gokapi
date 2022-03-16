var clipboard = new ClipboardJS('.btn');


var dropzoneObject;

Dropzone.options.uploaddropzone = {
  paramName: "file",
  maxFilesize: 102400, // 100 GB
  timeout: 7200000,
  createImageThumbnails: false,
  success: function (file, response) {
   addRow(response)
   this.removeFile(file);
 },
    init: function() {
      dropzoneObject = this;
        this.on("sending", function(file, xhr, formData){
                formData.append("allowedDownloads", document.getElementById("allowedDownloads").value);
                formData.append("expiryDays", document.getElementById("expiryDays").value);
                formData.append("password", document.getElementById("password").value);
                formData.append("isUnlimitedDownload", !document.getElementById("enableDownloadLimit").checked);
                formData.append("isUnlimitedTime", !document.getElementById("enableTimeLimit").checked);
        });
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


function checkBoxChanged(checkBox, correspondingInput) {
  let disable = !checkBox.checked;
  
  if (disable) {
  	 document.getElementById(correspondingInput).setAttribute("disabled", "");
  } else {
  	document.getElementById(correspondingInput).removeAttribute("disabled");
  }
  if (correspondingInput == "password" && disable) {
  	document.getElementById("password").value = "";
  }
}

function parseData(data) {
    if (!data) return {"Result":"error"};
    if (typeof data === 'object') return data;
    if (typeof data === 'string') return JSON.parse(data);

    return {"Result":"error"};
}

function addRow(jsonText) {
  let jsonObject = parseData(jsonText);
  if (jsonObject.Result != "OK") {
	alert("Failed to upload file!");
	location.reload();
	return;
  }
  let item = jsonObject.FileInfo;
  let table = document.getElementById("downloadtable");
  let row = table.insertRow(0);
  let cell1 = row.insertCell(0);
  let cell2 = row.insertCell(1);
  let cell3 = row.insertCell(2);
  let cell4 = row.insertCell(3);
  let cell5 = row.insertCell(4);
  let cell6 = row.insertCell(5);
  let lockIcon = "";
  
  if (item.PasswordHash != "") {
	lockIcon = " &#128274;";
  }
  cell1.innerText = item.Name;
  cell2.innerText = item.Size;
  if (item.UnlimitedDownloads) {
  	cell3.innerText = "Unlimited";
  } else {
  	cell3.innerText = item.DownloadsRemaining;
  }
  if (item.UnlimitedTime) {
  	cell4.innerText = "Unlimited";
  } else {
  	cell4.innerText = item.ExpireAtString;
  }
  cell5.innerHTML = '<a  target="_blank" style="color: inherit" href="'+jsonObject.Url+item.Id+'">'+jsonObject.Url+item.Id+'</a>'+lockIcon;

  let buttons = "<button type=\"button\" data-clipboard-text=\""+jsonObject.Url+item.Id+"\" class=\"copyurl btn btn-outline-light btn-sm\">Copy URL</button> ";
  if (item.HotlinkId != "") {
	buttons = buttons + '<button type="button" data-clipboard-text="'+jsonObject.HotlinkUrl+item.HotlinkId+'" class="copyurl btn btn-outline-light btn-sm">Copy Hotlink</button> ';
  } else {
	buttons = buttons + '<button type="button"class="copyurl btn btn-outline-light btn-sm disabled">Copy Hotlink</button> ';
  }
  buttons = buttons + "<button type=\"button\" class=\"btn btn-outline-light btn-sm\" onclick=\"window.location='./delete?id="+item.Id+"'\">Delete</button>";

  cell6.innerHTML = buttons;
  cell1.style.backgroundColor="green"
  cell2.style.backgroundColor="green"
  cell3.style.backgroundColor="green"
  cell4.style.backgroundColor="green"
  cell5.style.backgroundColor="green"
  cell6.style.backgroundColor="green"
}
