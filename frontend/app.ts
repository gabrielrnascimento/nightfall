let socket: WebSocket | null = null;

const statusEl = document.getElementById("status") as HTMLElement;
const logEl = document.getElementById("log") as HTMLElement;
const wsUrlEl = document.getElementById("wsUrl") as HTMLInputElement;
const connectBtn = document.getElementById("connectBtn") as HTMLButtonElement;
const disconnectBtn = document.getElementById(
  "disconnectBtn"
) as HTMLButtonElement;
const nameInput = document.getElementById("nameInput") as HTMLInputElement;
const roomInput = document.getElementById("roomInput") as HTMLInputElement;
const joinBtn = document.getElementById("joinBtn") as HTMLButtonElement;
const leaveBtn = document.getElementById("leaveBtn") as HTMLButtonElement;
const readyBtn = document.getElementById("readyBtn") as HTMLButtonElement;
const startBtn = document.getElementById("startBtn") as HTMLButtonElement;

function log(message: string) {
  const div = document.createElement("div");
  div.textContent = message;
  logEl.appendChild(div);
  logEl.scrollTop = logEl.scrollHeight;
}

function setStatus(text: string) {
  statusEl.innerHTML = "Status: <strong>" + text + "</strong>";
}

function connect() {
  const url = wsUrlEl.value.trim();
  if (!url) return;

  socket = new WebSocket(url);

  socket.addEventListener("open", () => {
    setStatus("connected");
    log("[open] Connected to " + url);
    connectBtn.disabled = true;
    disconnectBtn.disabled = false;
    joinBtn.disabled = false;
    leaveBtn.disabled = true;
  });

  socket.addEventListener("message", (event) => {
    log("[received] " + event.data);
    const message = JSON.parse(event.data);
    if (message.type === "joined") {
      joinBtn.disabled = true;
      leaveBtn.disabled = false;
      nameInput.disabled = true;
      roomInput.disabled = true;
      readyBtn.disabled = false;
      startBtn.disabled = false;
    }
    if (message.type === "left") {
      joinBtn.disabled = false;
      leaveBtn.disabled = true;
      nameInput.disabled = false;
      roomInput.disabled = false;
      startBtn.disabled = true;
      readyBtn.disabled = true;
    }
  });

  socket.addEventListener("close", (event) => {
    setStatus("disconnect");
    log("[close] Code: " + event.code + ", reason: " + event.reason);
    connectBtn.disabled = false;
    disconnectBtn.disabled = true;
    joinBtn.disabled = true;
    leaveBtn.disabled = true;
    readyBtn.disabled = true;
    nameInput.disabled = false;
    roomInput.disabled = false;
  });

  socket.addEventListener("error", (event) => {
    log("[error] See console for details");
    console.error("WebSocket error: ", event);
  });
}

connectBtn.onclick = function () {
  connect();
};

disconnectBtn.onclick = function () {
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close(1000, "Client disconnect");
  }
};

joinBtn.onclick = function () {
  const name = nameInput.value;
  const room = roomInput.value;

  if (!name || !room || !socket || socket.readyState !== WebSocket.OPEN) return;

  const joinMessage: JoinMessage = {
    type: "join",
    name,
    room,
  };

  socket.send(JSON.stringify(joinMessage));
  log("[sent] " + JSON.stringify(joinMessage));
};

leaveBtn.onclick = function () {
  const leaveMessage: LeaveMessage = {
    type: "leave",
  };
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify(leaveMessage));
  log("[sent] " + JSON.stringify(leaveMessage));
};

readyBtn.onclick = function () {
  const readyMessage: ReadyMessage = {
    type: "ready",
  };
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify(readyMessage));
  log("[sent] " + JSON.stringify(readyMessage));
};

startBtn.onclick = function () {
  const startMessage: StartMessage = {
    type: "start",
  };
  if (!socket || socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify(startMessage));
  log("[sent] " + JSON.stringify(startMessage));
};

window.addEventListener("load", function () {
  connect();
});

type JoinMessage = {
  type: string;
  name: string;
  room: string;
};

type LeaveMessage = {
  type: string;
};

type StartMessage = {
  type: string;
};

type ReadyMessage = {
  type: string;
};