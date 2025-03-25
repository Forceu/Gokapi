// This file contains JS code for the Logs view
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...

function filterLogs(tag) {
    if (tag == "all") {
        textarea.value = logContent;
    } else {
        textarea.value = logContent.split("\n").filter(line => line.includes("[" + tag + "]")).join("\n");
    }
    textarea.scrollTop = textarea.scrollHeight;
}

function deleteLogs(cutoff) {
    if (cutoff == "none") {
        return;
    }
    if (!confirm("Do you want to delete the selected logs?")) {
        document.getElementById('deleteLogs').selectedIndex = 0;
        return;
    }
    let timestamp = Math.floor(Date.now() / 1000)
    switch (cutoff) {
        case "all":
            timestamp = 0;
            break;
        case "2":
            timestamp = timestamp - 2 * 24 * 60 * 60;
            break;
        case "7":
            timestamp = timestamp - 7 * 24 * 60 * 60;
            break;
        case "14":
            timestamp = timestamp - 14 * 24 * 60 * 60;
            break;
        case "30":
            timestamp = timestamp - 30 * 24 * 60 * 60;
            break;
    }
    apiLogsDelete(timestamp)
        .then(data => {
            location.reload();
        })
        .catch(error => {
            alert("Unable to delete logs: " + error);
            console.error('Error:', error);
        });
}
