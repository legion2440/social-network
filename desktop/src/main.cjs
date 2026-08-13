'use strict';

const path = require('node:path');
const { app, BrowserWindow, ipcMain, Menu, Notification, shell } = require('electron');
const { createDesktopServer } = require('./proxy.cjs');
const { isAllowedOAuthURL, isOriginURL, mirrorSessionCookies } = require('./oauth.cjs');
const { loadSettings, normalizeServerURL, probeOrigin, saveSettings } = require('./settings.cjs');

const DEFAULT_ORIGIN = 'http://127.0.0.1:8080';
const DESKTOP_PARTITION = 'persist:loop';
let mainWindow = null;
let oauthWindow = null;
let oauthPollTimer = null;
let backendCheckTimer = null;
let settingsWindow = null;
let desktopServer = null;
let localOrigin = '';
let targetOrigin = '';
let webOrigin = '';
let serverSource = 'default';
let setupReason = 'first-run';
let setupError = '';
let firstRunResolve = null;
let firstRunReject = null;
let quitting = false;

function normalizedOrigin(value, fallback = '') {
  try {
    return normalizeServerURL(value, fallback);
  } catch (_error) {
    return fallback;
  }
}

function isExternalHTTPURL(value) {
  try {
    const parsed = new URL(value);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch (_error) {
    return false;
  }
}

function frontendDistPath() {
  if (app.isPackaged) return path.join(process.resourcesPath, 'frontend-dist');
  return path.resolve(__dirname, '..', '..', 'frontend', 'dist');
}

function sendNetworkStatus(online) {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('desktop:network-status', online === true);
  }
  if (online && serverSource !== 'env' && targetOrigin) {
    saveSettings(app.getPath('userData'), {
      serverURL: targetOrigin,
      lastSuccessfulAt: new Date().toISOString()
    }).catch(error => console.warn('desktop settings write failed:', error.message));
  }
}

function sendOAuthComplete() {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('desktop:oauth-complete');
  }
}

function openExternal(url) {
  if (!isExternalHTTPURL(url)) return;
  shell.openExternal(url).catch(() => {});
}

function clearOAuthWindow() {
  if (oauthPollTimer) {
    clearInterval(oauthPollTimer);
    oauthPollTimer = null;
  }
  oauthWindow = null;
}

async function completeDesktopOAuth(window, origin) {
  if (!window || window.isDestroyed()) return false;
  const currentURL = window.webContents.getURL();
  if (!isOriginURL(currentURL, origin)) return false;

  let authenticated = false;
  try {
    authenticated = await window.webContents.executeJavaScript(
      "fetch('/api/auth/me',{method:'GET',credentials:'include',headers:{Accept:'application/json'}})" +
      ".then(response=>response.status===200).catch(()=>false)",
      true
    );
  } catch (_error) {
    return false;
  }
  if (!authenticated) return false;

  try {
    await mirrorSessionCookies(window.webContents.session, origin, localOrigin);
  } catch (error) {
    console.error('desktop OAuth cookie handoff failed:', error);
    return false;
  }

  sendOAuthComplete();
  if (!window.isDestroyed()) window.close();
  return true;
}

async function startGitHubOAuth(origin) {
  if (oauthWindow && !oauthWindow.isDestroyed()) {
    oauthWindow.focus();
    return true;
  }

  const window = new BrowserWindow({
    width: 720,
    height: 820,
    minWidth: 520,
    minHeight: 620,
    parent: mainWindow || undefined,
    show: false,
    title: 'Sign in with GitHub',
    autoHideMenuBar: true,
    webPreferences: {
      partition: DESKTOP_PARTITION,
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true
    }
  });
  oauthWindow = window;

  window.webContents.setWindowOpenHandler(({ url }) => {
    if (isAllowedOAuthURL(url, origin)) window.loadURL(url).catch(() => {});
    return { action: 'deny' };
  });
  window.webContents.on('will-navigate', (event, url) => {
    if (isAllowedOAuthURL(url, origin)) return;
    event.preventDefault();
  });
  window.once('ready-to-show', () => window.show());
  window.on('closed', clearOAuthWindow);

  let checking = false;
  oauthPollTimer = setInterval(async () => {
    if (checking || window.isDestroyed()) return;
    checking = true;
    try {
      await completeDesktopOAuth(window, origin);
    } finally {
      checking = false;
    }
  }, 500);

  try {
    await window.loadURL(origin + '/api/auth/oauth/github/start?next=/');
    return true;
  } catch (error) {
    if (!window.isDestroyed()) window.close();
    throw error;
  }
}

function createWindow() {
  const window = new BrowserWindow({
    width: 1200,
    height: 820,
    minWidth: 760,
    minHeight: 560,
    show: false,
    title: 'Loop',
    backgroundColor: '#f7f7fb',
    autoHideMenuBar: true,
    webPreferences: {
      partition: DESKTOP_PARTITION,
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true
    }
  });

  window.webContents.setWindowOpenHandler(({ url }) => {
    if (url.startsWith(localOrigin + '/')) return { action: 'allow' };
    openExternal(url);
    return { action: 'deny' };
  });

  window.webContents.on('will-navigate', (event, url) => {
    if (url === localOrigin || url.startsWith(localOrigin + '/')) return;
    event.preventDefault();
    openExternal(url);
  });

  window.once('ready-to-show', () => window.show());
  window.on('closed', () => {
    if (mainWindow === window) mainWindow = null;
  });
  return window;
}

function createSettingsWindow(reason) {
  if (settingsWindow && !settingsWindow.isDestroyed()) {
    setupReason = reason || setupReason;
    settingsWindow.focus();
    return settingsWindow;
  }
  setupReason = reason || 'manual';
  const window = new BrowserWindow({
    width: 620,
    height: 590,
    minWidth: 500,
    minHeight: 520,
    resizable: true,
    show: false,
    title: 'Loop server settings',
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'setup-preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true
    }
  });
  settingsWindow = window;
  window.once('ready-to-show', () => window.show());
  window.on('closed', () => {
    settingsWindow = null;
    if (firstRunResolve && !quitting) {
      const reject = firstRunReject;
      firstRunResolve = null;
      firstRunReject = null;
      if (reject) reject(new Error('Server setup was cancelled'));
    }
  });
  window.loadFile(path.join(__dirname, 'setup.html')).catch(error => {
    console.error('desktop server settings failed to load:', error);
    window.close();
  });
  return window;
}

async function resolveInitialServer() {
  const envValue = String(process.env.SOCIAL_NETWORK_URL || '').trim();
  if (envValue) {
    const origin = normalizeServerURL(envValue);
    serverSource = 'env';
    return origin;
  }

  const saved = await loadSettings(app.getPath('userData'));
  if (saved && saved.serverURL) {
    serverSource = 'settings';
    return saved.serverURL;
  }

  serverSource = 'settings';
  setupReason = 'first-run';
  setupError = '';
  return new Promise((resolve, reject) => {
    firstRunResolve = resolve;
    firstRunReject = reject;
    createSettingsWindow('first-run');
  });
}

function installMenu() {
  const template = [];
  if (process.platform === 'darwin') {
    template.push({
      label: app.name,
      submenu: [
        { label: 'Server Settings…', click: () => createSettingsWindow('manual') },
        { type: 'separator' },
        { role: 'quit' }
      ]
    });
  } else {
    template.push({
      label: 'Settings',
      submenu: [
        { label: 'Server…', click: () => createSettingsWindow('manual') }
      ]
    });
  }
  template.push({
    label: 'View',
    submenu: [
      { role: 'reload' },
      { role: 'togglefullscreen' }
    ]
  });
  Menu.setApplicationMenu(Menu.buildFromTemplate(template));
}

function registerSettingsIPC() {
  ipcMain.handle('desktop:server-settings:get', async () => {
    const saved = await loadSettings(app.getPath('userData'));
    const locked = String(process.env.SOCIAL_NETWORK_URL || '').trim() !== '';
    return {
      serverURL: locked
        ? normalizedOrigin(process.env.SOCIAL_NETWORK_URL, DEFAULT_ORIGIN)
        : (targetOrigin || saved && saved.serverURL || DEFAULT_ORIGIN),
      locked,
      reason: setupReason,
      error: setupError
    };
  });

  ipcMain.handle('desktop:server-settings:connect', async (_event, value) => {
    if (String(process.env.SOCIAL_NETWORK_URL || '').trim()) {
      return { ok: false, error: 'Server address is controlled by SOCIAL_NETWORK_URL.' };
    }
    let origin;
    try {
      origin = normalizeServerURL(value);
    } catch (error) {
      return { ok: false, error: error.message || 'Enter a valid server address.' };
    }
    const reachable = await probeOrigin(origin);
    if (!reachable) {
      setupError = 'Could not reach the social-network server at this address.';
      return { ok: false, error: setupError };
    }
    setupError = '';
    await saveSettings(app.getPath('userData'), {
      serverURL: origin,
      lastSuccessfulAt: new Date().toISOString()
    });
    if (firstRunResolve) {
      const resolve = firstRunResolve;
      firstRunResolve = null;
      firstRunReject = null;
      targetOrigin = origin;
      if (settingsWindow && !settingsWindow.isDestroyed()) settingsWindow.close();
      resolve(origin);
      return { ok: true, serverURL: origin };
    }
    if (origin !== targetOrigin) {
      app.relaunch();
      app.exit(0);
    } else if (settingsWindow && !settingsWindow.isDestroyed()) {
      settingsWindow.close();
    }
    return { ok: true, serverURL: origin };
  });

  ipcMain.handle('desktop:server-settings:close', async () => {
    if (settingsWindow && !settingsWindow.isDestroyed()) settingsWindow.close();
    return true;
  });
}

app.setAppUserModelId('com.legion2440.loop');

app.whenReady().then(async () => {
  registerSettingsIPC();
  installMenu();

  targetOrigin = await resolveInitialServer();
  webOrigin = normalizedOrigin(process.env.SOCIAL_NETWORK_WEB_URL, targetOrigin);

  desktopServer = createDesktopServer({
    distDir: frontendDistPath(),
    targetOrigin,
    cacheDir: path.join(app.getPath('userData'), 'offline-cache'),
    onNetworkStatus: sendNetworkStatus
  });
  const listener = await desktopServer.listen();
  localOrigin = listener.origin;

  ipcMain.handle('desktop:open-registration', async () => {
    const url = new URL(webOrigin + '/');
    url.searchParams.set('register', '1');
    await shell.openExternal(url.toString());
    return true;
  });

  ipcMain.handle('desktop:start-github-oauth', async () => startGitHubOAuth(targetOrigin));

  ipcMain.handle('desktop:notify', (_event, payload) => {
    if (!Notification.isSupported()) return false;
    const title = String(payload && payload.title || 'New message').slice(0, 120);
    const body = String(payload && payload.body || '').slice(0, 300);
    new Notification({ title, body }).show();
    return true;
  });

  ipcMain.handle('desktop:set-connectivity', async (_event, online) => {
    desktopServer.setClientOnline(online === true);
    if (online === true) await desktopServer.checkBackend();
    return desktopServer.isOnline();
  });

  mainWindow = createWindow();
  await mainWindow.loadURL(localOrigin);
  desktopServer.checkBackend().catch(() => {});
  backendCheckTimer = setInterval(() => {
    if (desktopServer) desktopServer.checkBackend().catch(() => {});
  }, 3000);

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      mainWindow = createWindow();
      mainWindow.loadURL(localOrigin).catch(() => {});
    }
  });
}).catch(error => {
  if (!quitting) console.error(error);
  app.quit();
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

app.on('before-quit', () => {
  quitting = true;
  if (backendCheckTimer) clearInterval(backendCheckTimer);
  if (oauthPollTimer) clearInterval(oauthPollTimer);
  if (oauthWindow && !oauthWindow.isDestroyed()) oauthWindow.destroy();
  if (settingsWindow && !settingsWindow.isDestroyed()) settingsWindow.destroy();
  if (desktopServer) desktopServer.close().catch(() => {});
});
