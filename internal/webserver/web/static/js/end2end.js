function getE2EInfo() {
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "./e2eInfo?action=get", false);
    xhr.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                console.log(xhr.response);
            } else {
                console.log("TODO: Could not get e2e info!");
            }
        }
    };

    xhr.send();
}

function storeE2EInfo(data) {
    var xhr = new XMLHttpRequest();
    xhr.open("POST", "./e2eInfo?action=store", false);
    xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
    
    xhr.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                console.log(xhr.response);
            } else {
                console.log("TODO: Could not store e2e info!");
            }
        }
    };
    let formData = new FormData();
    formData.append("info", data);
    xhr.send(urlencodeFormData(formData));
}
