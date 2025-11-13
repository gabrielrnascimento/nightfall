let socket = null;

const statusEl = document.getElementById("status");
const logEl = document.getElementById("log");
const wsUrlEl = document.getElementById("wsUrl");
const connectBtn = document.getElementById("connectBtn");
const disconnectBtn = document.getElementById("disconnectBtn");
const messageInput = document.getElementById("messageInput");
const sendBtn = document.getElementById("sendBtn");

function log(message) {
  const div = document.createElement("div");
  div.textContent = message;
  logEl.appendChild(div);
  logEl.scrollTop = logEl.scrollHeight;
}

function setStatus(text) {
  statusEl.innerHTML = "Status: <string>" + text + "</strong>";
}

connectBtn.onclick = function () {
  const url = wsUrlEl.value.trim();
  if (!url) return;

  socket = new WebSocket(url);

  socket.addEventListener("open", () => {
    setStatus("connected");
    log("[open] Connected to " + url);
    connectBtn.disabled = true;
    disconnectBtn.disabled = false;
    sendBtn.disabled = false;
  });

  socket.addEventListener("message", (event) => {
    log("[received] " + event.data);
  });

  socket.addEventListener("close", (event) => {
    setStatus("disconnect");
    log("[close] Code: " + event.code + ", reason: " + event.reason);
    connectBtn.disabled = false;
    disconnectBtn.disabled = true;
    sendBtn.disabled = true;
  });

  socket.addEventListener("error", (event) => {
    log("[error] See console for details");
    console.error("WebSocket error: ", error);
  });
};

disconnectBtn.onclick = function () {
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close(1000, "Client disconnect");
  }
};

sendBtn.onclick = function () {
  const msg = messageInput.value;
  if (!msg || !socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(msg);
  log("[sent] " + msg);
  messageInput.value = "";
};

messageInput.addEventListener("keydown", function (e) {
  if (e.key === "Enter") sendBtn.click();
});
