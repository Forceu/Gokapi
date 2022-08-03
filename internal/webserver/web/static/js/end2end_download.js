function parseHashValue(id) {
    let key = localStorage.getItem("key-" + id);
    let filename = localStorage.getItem("fn-" + id);

    if (key === null || filename === null) {
        hash = window.location.hash.substr(1);
        if (hash.length < 50) {
            redirectToE2EError();
            return;
        }
        let info;
        try {
            let infoJson = atob(hash);
            info = JSON.parse(infoJson)
        } catch (err) {
            redirectToE2EError();
            return;
        }
        if (!isCorrectJson(info)) {
            redirectToE2EError();
            return;
        }
        localStorage.setItem("key-" + id, info.c);
        localStorage.setItem("fn-" + id, info.f);
    }
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
