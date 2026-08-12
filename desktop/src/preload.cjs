'use strict';

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('loopDesktop', {
  isDesktop: true,
  openRegistration: () => ipcRenderer.invoke('desktop:open-registration'),
  startGitHubOAuth: () => ipcRenderer.invoke('desktop:start-github-oauth'),
  notify: payload => ipcRenderer.invoke('desktop:notify', payload || {}),
  setConnectivity: online => ipcRenderer.invoke('desktop:set-connectivity', online === true),
  onNetworkStatus: callback => {
    if (typeof callback !== 'function') return () => {};
    const listener = (_event, online) => callback(online === true);
    ipcRenderer.on('desktop:network-status', listener);
    return () => ipcRenderer.removeListener('desktop:network-status', listener);
  },
  onOAuthComplete: callback => {
    if (typeof callback !== 'function') return () => {};
    const listener = () => callback();
    ipcRenderer.on('desktop:oauth-complete', listener);
    return () => ipcRenderer.removeListener('desktop:oauth-complete', listener);
  }
});
