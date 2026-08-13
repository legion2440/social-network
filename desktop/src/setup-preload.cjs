'use strict';

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('loopSetup', {
  getState: () => ipcRenderer.invoke('desktop:server-settings:get'),
  connect: serverURL => ipcRenderer.invoke('desktop:server-settings:connect', String(serverURL || '')),
  close: () => ipcRenderer.invoke('desktop:server-settings:close')
});
