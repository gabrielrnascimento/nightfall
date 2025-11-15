let socket: WebSocket | null = null;

const statusEl = document.getElementById("status") as HTMLElement;
const logEl = document.getElementById("log") as HTMLElement;
const wsUrlEl = document.getElementById("wsUrl") as HTMLInputElement;
const connectBtn = document.getElementById("connectBtn") as HTMLButtonElement;
const disconnectBtn = document.getElementById(
  "disconnectBtn"
) as HTMLButtonElement;
const messageInput = document.getElementById(
  "messageInput"
) as HTMLInputElement;
const sendBtn = document.getElementById("sendBtn") as HTMLButtonElement;

function log(message: string) {
  const div = document.createElement("div");
  div.textContent = message;
  logEl.appendChild(div);
  logEl.scrollTop = logEl.scrollHeight;
}

function setStatus(text: string) {
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
    console.error("WebSocket error: ", event);
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
