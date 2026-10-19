import { migrateCommandOptions } from './migrateOptions';
import { PanelOptions } from './types';

const migrate = (options: PanelOptions, targets?: Array<{ refId: string; command: string }>) =>
    migrateCommandOptions({
        id: 1,
        type: 'jaops-commanding-panel',
        options,
        targets,
        fieldConfig: { defaults: { custom: {} }, overrides: [] },
    }) as PanelOptions;

describe('migrateCommandOptions', () => {
    it('preserves old commanding form settings and their button state', () => {
        const legacyForm = { label: 'Send', arguments: { mode: 2 }, comment: 'existing' };
        const options = {
            commandForms: { 'Start0': legacyForm, 'Stop1': { label: 'Stop' } },
            dualButtonStates: { 'Start0': 'on' },
        } as unknown as PanelOptions;
        const result = migrate(options, [{ refId: 'A', command: '/ops/Start' }]);
        expect(result.commandForms['command-0']).toEqual({ ...legacyForm, commandName: '/ops/Start' });
        expect(result.commandForms['command-1']).toEqual({ label: 'Stop', commandName: 'Stop' });
        expect(result.dualButtonStates?.['command-0']).toBe('on');
        expect(result.commandForms['Start0']).toBeUndefined();
    });

    it('preserves old variable panel settings without inventing a command', () => {
        const result = migrate({ commandForms: { undefined0: { variableToSet: 'target' } } } as unknown as PanelOptions);
        expect(result.commandForms['command-0'].variableToSet).toBe('target');
        expect(result.commandForms['command-0'].commandName).toBe('');
    });

    it('keeps settings already using current keys', () => {
        const options = { commandForms: { 'command-0': { label: 'current' } } } as unknown as PanelOptions;
        expect(migrate(options).commandForms).toEqual(options.commandForms);
    });
});
