'use strict';

const path = require('node:path');
const { app, BrowserWindow, ipcMain, Notification, shell } = require('electron');
const { createDesktopServer } = require('./proxy.cjs');
const { isOriginURL, mirrorSessionCookies } = require('./oauth.cjs');

const DEFAULT_ORIGIN = 'http://127.0.0.1:8080';
const DESKTOP_PARTITION = 'persist:loop';
let mainWindow = null;
let oauthWindow = null;
let oauthPollTimer = null;
let desktopServer = null;
let localOrigin = '';

function normalizedOrigin(value, fallback) {
  try {
    const parsed = new URL(value || fallback);
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return fallback;
    return parsed.origin;
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

async function completeDesktopOAuth(window, targetOrigin) {
  if (!window || window.isDestroyed()) return false;
  const currentURL = window.webContents.getURL();
  if (!isOriginURL(currentURL, targetOrigin)) return false;

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
    await mirrorSessionCookies(window.webContents.session, targetOrigin, localOrigin);
  } catch (error) {
    console.error('desktop OAuth cookie handoff failed:', error);
    return false;
  }

  sendOAuthComplete();
  if (!window.isDestroyed()) window.close();
  return true;
}

async function startGitHubOAuth(targetOrigin) {
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
    if (isExternalHTTPURL(url)) {
      window.loadURL(url).catch(() => {});
    }
    return { action: 'deny' };
  });
  window.once('ready-to-show', () => window.show());
  window.on('closed', clearOAuthWindow);

  let checking = false;
  oauthPollTimer = setInterval(async () => {
    if (checking || window.isDestroyed()) return;
    checking = true;
    try {
      await completeDesktopOAuth(window, targetOrigin);
    } finally {
      checking = false;
    }
  }, 500);

  try {
    await window.loadURL(targetOrigin + '/api/auth/oauth/github/start?next=/');
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

app.setAppUserModelId('com.legion2440.loop');

app.whenReady().then(async () => {
  const targetOrigin = normalizedOrigin(process.env.SOCIAL_NETWORK_URL, DEFAULT_ORIGIN);
  const webOrigin = normalizedOrigin(process.env.SOCIAL_NETWORK_WEB_URL, targetOrigin);

  desktopServer = createDesktopServer({
    distDir: frontendDistPath(),
    targetOrigin,
    cacheDir: path.join(app.getPath('userData'), 'offline-cache'),
    onNetworkStatus: sendNetworkStatus
  });
  const listener = await desktopServer.listen();
  localOrigin = listener.origin;

  ipcMain.handle('desktop:open-registration', async () => {
    await shell.openExternal(webOrigin);
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

  ipcMain.handle('desktop:set-connectivity', (_event, online) => {
    desktopServer.setClientOnline(online === true);
    return desktopServer.isOnline();
  });

  mainWindow = createWindow();
  await mainWindow.loadURL(localOrigin);

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      mainWindow = createWindow();
      mainWindow.loadURL(localOrigin).catch(() => {});
    }
  });
}).catch(error => {
  console.error(error);
  app.quit();
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});

app.on('before-quit', () => {
  if (oauthPollTimer) clearInterval(oauthPollTimer);
  if (oauthWindow && !oauthWindow.isDestroyed()) oauthWindow.destroy();
  if (desktopServer) desktopServer.close().catch(() => {});
});
