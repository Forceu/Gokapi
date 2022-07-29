function getE2EInfo() {
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "./e2eInfo?action=get", false);
    xhr.onreadystatechange = function() {
        if (this.readyState == 4) {
            if (this.status == 200) {
                let err = GokapiE2EInfoParse(xhr.response); //TODO
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
    console.log("sending: "+data);
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
        console.log(err);
        //TODO
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
