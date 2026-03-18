/**
 * sse-worker.js  –  SharedWorker for SSE fan-out
 *
 * One SSE connection is shared across all tabs from the same origin.
 * Each tab connects via a MessagePort. Messages are broadcast to all ports.
 * The SSE connection is lazily opened on first tab connect and closed when
 * the last tab disconnects, so no dangling connections are left behind.
 */

// Will be loaded from [root]/js/min
const SSE_URL = "../../uploadStatus";
const RECONNECT_DELAY_MS = 5000;

/** @type {MessagePort[]} */
const ports = [];
let source = null;
let reconnectTimer = null;

function broadcast(data) {
    for (const port of ports) {
        try {
            port.postMessage(data);
        } catch (_) {
            // port may have gone away; it will be cleaned up via "close"
        }
    }
}


const DEBUG_OUTPUT = false;

/**
 * broadcastLog forwards a log entry to all connected tabs so it appears in
 * the page DevTools console. This is necessary because console.log() inside a
 * SharedWorker is only visible in the worker's own inspector context
 * (chrome://inspect/#workers or about:debugging in Firefox), not in the tab.
 * @param {"log"|"warn"|"error"} level
 * @param {string} message
 * @param {Object} [detail]
 */
function broadcastLog(level, message, detail) {
    if (level == "log" && !DEBUG_OUTPUT) {
    	return;
    }
    broadcast({ type: "log", level, message, detail: detail || null });
}

function connect() {
    if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }

    broadcastLog("log", "[sse-worker] Opening SSE connection", { url: SSE_URL, ports: ports.length });
    source = new EventSource(SSE_URL);

    source.onopen = () => {
        broadcastLog("log", "[sse-worker] SSE connection established", { url: SSE_URL });
    };

    source.onmessage = (event) => {
        broadcast({ type: "message", data: event.data });
    };

    source.addEventListener("ping", () => {
        // keep-alive ping from server – nothing to forward to tabs
    });

    source.onerror = (event) => {
        const detail = {
            readyState: source.readyState,  // 0=CONNECTING, 1=OPEN, 2=CLOSED
            url: SSE_URL,
            ports: ports.length,
        };
        broadcastLog("error", "[sse-worker] SSE connection error", detail);
        source.close();
        source = null;
        broadcast({ type: "error", detail });
        // Only schedule reconnect if there are still tabs listening
        if (ports.length > 0) {
            broadcastLog("log", `[sse-worker] Reconnecting in ${RECONNECT_DELAY_MS}ms`, { ports: ports.length });
            reconnectTimer = setTimeout(connect, RECONNECT_DELAY_MS);
        }
    };
}

function removePort(port) {
    const idx = ports.indexOf(port);
    if (idx !== -1) {
        ports.splice(idx, 1);
    }
    broadcastLog("log", "[sse-worker] Port removed", { remainingPorts: ports.length });
    // Close the SSE connection when no tabs remain to free the browser slot
    if (ports.length === 0) {
        if (reconnectTimer !== null) {
            clearTimeout(reconnectTimer);
            reconnectTimer = null;
        }
        if (source !== null) {
            broadcastLog("log", "[sse-worker] Last port removed, closing SSE connection");
            source.close();
            source = null;
        }
    }
}

/**
 * shutdown closes the SSE connection and notifies every connected tab.
 * Used when any tab triggers a logout, since the session is now invalid
 * for all tabs — not just the one the user clicked logout in.
 */
function shutdown() {
    broadcastLog("log", "[sse-worker] Shutdown requested, closing all ports", { totalPorts: ports.length });
    if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
    }
    if (source !== null) {
        source.close();
        source = null;
    }
    // Notify all tabs so they can react (e.g. redirect to login)
    broadcast({ type: "shutdown" });
    ports.length = 0;
}

// Invoked once per tab that connects to this SharedWorker
self.onconnect = (event) => {
    const port = event.ports[0];
    ports.push(port);
    broadcastLog("log", "[sse-worker] Port added", { totalPorts: ports.length });

    port.onmessage = (msg) => {
        if (msg.data && msg.data.type === "close") {
            removePort(port);
        } else if (msg.data && msg.data.type === "shutdown") {
            shutdown();
        }
    };

    port.addEventListener("close", () => removePort(port));

    port.start();

    // Lazily open the SSE connection on first tab
    if (source === null && reconnectTimer === null) {
        connect();
    }
};
