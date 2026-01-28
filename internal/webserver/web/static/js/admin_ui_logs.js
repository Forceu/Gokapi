// This file contains JS code for the Logs view
// All files named admin_*.js will be merged together and minimised by calling
// go generate ./...

function filterLogs(tag) {
    const textarea = document.getElementById('logviewer');
    if (tag == "all") {
        textarea.value = logContent;
    } else {
        textarea.value = logContent.split("\n").filter(line => line.includes("[" + tag + "]")).join("\n");
    }
    textarea.scrollTop = textarea.scrollHeight;
}

function setMemoryUsage(used, total) {
    insertReadableSizeTwoOutputs(total, 'totalMemory', 'memoryUnit');
    let unit = document.getElementById('memoryUnit').innerText;
    insertReadableSizeForcedUnit(used, 'usedMemory', unit);
}

function setDiskUsage(used, total) {
    insertReadableSizeTwoOutputs(total, 'totalDisk', 'diskUnit');
    let unit = document.getElementById('diskUnit').innerText;
    insertReadableSizeForcedUnit(used, 'usedDisk', unit);
}


function formatDuration(seconds) {
    const units = [{
            label: "y",
            value: 31536000
        },
        {
            label: "d",
            value: 86400
        },
        {
            label: "h",
            value: 3600
        },
        {
            label: "m",
            value: 60
        },
        {
            label: "s",
            value: 1
        },
    ];

    let startIndex = units.findIndex(unit => seconds >= unit.value);

    // If everything is below 1 minute, force start at minutes
    if (startIndex === -1 || units[startIndex].label === "s") {
        startIndex = units.findIndex(u => u.label === "m");
    }

    const first = units[startIndex];
    const second = units[startIndex + 1];

    const firstAmount = Math.floor(seconds / first.value);
    const remainder = seconds % first.value;
    const secondAmount = Math.floor(remainder / second.value);

    return `${firstAmount}${first.label} ${secondAmount}${second.label}`;
}

function addUptime() {
    if (currentUptime > 3600) {
        return;
    }
    setTimeout(() => {
        currentUptime = currentUptime + 1;
        document.getElementById('uptime').innerText = formatDuration(currentUptime);
        addUptime();
    }, 1000);
}

function setPercentageBar(id, num1, num2) {
    let percentage = num1;
    if (num2 !== undefined) {
        percentage = ((num1 / num2) * 100);
    }

    const bar = document.getElementById(id);

    bar.classList.remove("bg-success");
    bar.classList.remove("bg-warning");
    bar.classList.remove("bg-danger");

    if (percentage < 70) {
        bar.classList.add("bg-success");
    }
    if (percentage >= 70 && percentage < 90) {
        bar.classList.add("bg-warning");
    }
    if (percentage >= 90) {
        bar.classList.add("bg-danger");
    }
    bar.style.width = percentage + '%';
}


async function loadLogs(timestamp) {
    const textarea = document.getElementById('logviewer');

    try {
        const data = await apiLogGet(timestamp);
        lastLogUpdate = data.timestamp;
        let doScroll = true;
        if (timestamp != 0) {
            if (data.logEntries == "") {
                return;
            }
            doScroll = allowScroll();
            logContent = logContent + data.logEntries;
        } else {
            logContent = data.logEntries;
        }
        filterLogs(document.getElementById('logFilter').value);
        if (doScroll) {
            textarea.scrollTop = textarea.scrollHeight;
        }
    } catch (error) {
        lastLogUpdate = 0;
        console.error("Failed to load logs:", error);
        textarea.value = "Error loading logs. See console for details.";
    }
}


async function loadStatus() {
    try {
        const data = await apiLogSystemStatus();

        currentUptime = data.uptime;
        document.getElementById('labelCpu').innerText = data.cpuLoad + '%';
        document.getElementById('labelActiveFiles').innerText = data.activeFiles;
        setPercentageBar("barCpu", data.cpuLoad);
        setPercentageBar("barDisk", data.diskUsagePercentage);
        setPercentageBar("barMemory", data.memoryUsagePercentage);
        setMemoryUsage(data.memoryUsed, data.memoryTotal);
        setDiskUsage(data.diskUsed, data.diskTotal);
        insertReadableSizeTwoOutputs(data.dataServed, 'totalTraffic', 'totalTrafficUnit');
    } catch (error) {
        console.error("Failed to server status:", error);
    }
}

async function pollInfo() {
    firstStart = true;
    while (true) {
        await loadLogs(lastLogUpdate);
        if (firstStart) {
            firstStart = false;
        } else {
            await loadStatus();
        }
        await new Promise(r => setTimeout(r, POLL_INTERVAL_S * 1000));
    }
}

function allowScroll() {
    const textarea = document.getElementById('logviewer');
    return textarea.scrollTop + textarea.clientHeight >= textarea.scrollHeight - 5;
}


function deleteLogs() {
    const delSelector = document.getElementById("deleteLogsSel");
    if (!delSelector) {
        return;
    }
    const cutoff = delSelector.value;
    if (cutoff == "none" || cutoff == "") {
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
        default:
            return;
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
