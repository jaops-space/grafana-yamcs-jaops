import pluginJson from './plugin.json';

export const PLUGIN_BASE_URL = `/a/${pluginJson.id}`;

export enum ROUTES {
    HowToUse = 'how-to-use',
    Commanding = 'commanding-setup',
    Image = 'image-panel-setup',
    VariableSetup = 'variable-setup',
}
