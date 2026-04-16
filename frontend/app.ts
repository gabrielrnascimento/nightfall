// ── State ────────────────────────────────────────────────────────────────────

let ws: WebSocket | null = null;
let myName = '';
let currentRoom = '';
const players = new Map<string, boolean>(); // name → isReady

// ── Element references ────────────────────────────────────────────────────────

const entryScreen  = document.getElementById('entry-screen')  as HTMLElement;
const lobbyScreen  = document.getElementById('lobby-screen')  as HTMLElement;
const gameOverlay  = document.getElementById('game-overlay')  as HTMLElement;
const nameInput    = document.getElementById('nameInput')     as HTMLInputElement;
const roomInput    = document.getElementById('roomInput')     as HTMLInputElement;
const enterBtn     = document.getElementById('enterBtn')      as HTMLButtonElement;
const errorMsg     = document.getElementById('errorMsg')      as HTMLElement;
const roomNameEl   = document.getElementById('roomName')      as HTMLElement;
const readyCounterEl = document.getElementById('readyCounter') as HTMLElement;
const playerListEl = document.getElementById('playerList')    as HTMLElement;
const readyBtn     = document.getElementById('readyBtn')      as HTMLButtonElement;
const startBtn     = document.getElementById('startBtn')      as HTMLButtonElement;

// ── Screen management ─────────────────────────────────────────────────────────

function showScreen(id: 'entry' | 'lobby'): void {
  entryScreen.classList.toggle('active', id === 'entry');
  lobbyScreen.classList.toggle('active', id === 'lobby');
}

// ── Rendering ─────────────────────────────────────────────────────────────────

function renderPlayers(): void {
  if (players.size === 0) {
    playerListEl.innerHTML = '<p class="waiting-msg">aguardando jogadores...</p>';
    return;
  }

  playerListEl.innerHTML = '';
  players.forEach((isReady, name) => {
    const row = document.createElement('div');
    row.className = 'player-row';
    if (isReady) row.classList.add('ready');
    if (name === myName) row.classList.add('is-me');

    const nameEl = document.createElement('span');
    nameEl.className = 'player-name';
    nameEl.textContent = name === myName ? `${name} (você)` : name;

    const statusEl = document.createElement('span');
    statusEl.className = 'player-status';
    statusEl.textContent = isReady ? 'PRONTO' : '—';

    row.appendChild(nameEl);
    row.appendChild(statusEl);
    playerListEl.appendChild(row);
  });
}

function updateCounter(): void {
  const total = players.size;
  const ready = Array.from(players.values()).filter(Boolean).length;
  readyCounterEl.textContent = `${ready} / ${total} prontos`;
}

// ── WebSocket message handler ─────────────────────────────────────────────────

type IncomingMessage =
  | { type: 'joined';       room: string }
  | { type: 'user_joined';  name: string }
  | { type: 'user_left';    name: string }
  | { type: 'user_ready';   name: string }
  | { type: 'game_started' }
  | { type: 'left';         room: string };

function handleMessage(raw: string): void {
  const msg = JSON.parse(raw) as IncomingMessage;

  switch (msg.type) {
    case 'joined':
      currentRoom = msg.room;
      players.clear();
      players.set(myName, false);
      roomNameEl.textContent = currentRoom.toUpperCase();
      updateCounter();
      renderPlayers();
      showScreen('lobby');
      break;

    case 'user_joined':
      // Skip self — already added on 'joined'
      if (msg.name === myName) break;
      players.set(msg.name, false);
      updateCounter();
      renderPlayers();
      break;

    case 'user_left':
      players.delete(msg.name);
      updateCounter();
      renderPlayers();
      break;

    case 'user_ready':
      players.set(msg.name, true);
      updateCounter();
      renderPlayers();
      break;

    case 'game_started':
      gameOverlay.classList.remove('hidden');
      break;
  }
}

// ── Entry screen ──────────────────────────────────────────────────────────────

enterBtn.addEventListener('click', () => {
  const name = nameInput.value.trim();
  const room = roomInput.value.trim();

  if (!name || !room) {
    errorMsg.textContent = 'preencha nome e sala';
    return;
  }

  errorMsg.textContent = '';
  enterBtn.disabled = true;
  enterBtn.textContent = 'CONECTANDO...';

  ws = new WebSocket('ws://127.0.0.1:3001');

  ws.onopen = () => {
    myName = name;
    ws!.send(JSON.stringify({ type: 'join', name, room }));
  };

  ws.onmessage = (event) => handleMessage(event.data as string);

  ws.onerror = () => {
    errorMsg.textContent = 'erro: não foi possível conectar';
    enterBtn.disabled = false;
    enterBtn.textContent = 'ENTRAR';
  };

  ws.onclose = () => {
    // If we were in the lobby, return to entry
    if (lobbyScreen.classList.contains('active')) {
      showScreen('entry');
      players.clear();
    }
    enterBtn.disabled = false;
    enterBtn.textContent = 'ENTRAR';
  };
});

// ── Lobby actions ─────────────────────────────────────────────────────────────

readyBtn.addEventListener('click', () => {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify({ type: 'ready' }));
  readyBtn.disabled = true;
});

startBtn.addEventListener('click', () => {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify({ type: 'start' }));
});
