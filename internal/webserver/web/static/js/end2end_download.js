function parseHashValue(id) {
    let key = sessionStorage.getItem("key-" + id);
    let filename = sessionStorage.getItem("fn-" + id);

    if (key === null || filename === null) {
        hash = window.location.hash.substr(1);
        if (hash.length < 50) {
            redirectToE2EError();
            return;
        }
        let info;
        try {
            let infoJson = b64ToUtf8(hash);
            info = JSON.parse(infoJson)
        } catch (err) {
            redirectToE2EError();
            return;
        }
        if (!isCorrectJson(info)) {
            redirectToE2EError();
            return;
        }
        sessionStorage.setItem("key-" + id, info.c);
        sessionStorage.setItem("fn-" + id, info.f);
    }
}

function b64ToUtf8(str) {
  let bytes = Uint8Array.from(atob(str), c => c.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

function isCorrectJson(input) {
    return (input.f !== undefined &&
        input.c !== undefined &&
        typeof input.f === 'string' &&
        typeof input.c === 'string' &&
        input.f != "" &&
        input.c != "");
}

function redirectToE2EError() {
    window.location = "./error?e2e";
}
