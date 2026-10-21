import { migrateCommandOptions } from './migrateOptions';
import { PanelOptions } from './types';

type Target = { refId: string; command?: string };

const migrate = (options: Partial<PanelOptions>, targets: Target[] = [], type = 'jaops-commanding-panel') =>
    migrateCommandOptions({
        id: 1,
        type,
        options: options as PanelOptions,
        targets,
        fieldConfig: { defaults: { custom: {} }, overrides: [] },
    }) as PanelOptions;

describe('migrateCommandOptions', () => {
    it('preserves old commanding form settings and button state using target commands', () => {
        const legacyForm = { label: 'Send', arguments: { mode: 2 }, comment: 'existing' };
        const result = migrate(
            {
                commandForms: { '/ops/Start0': legacyForm, 'Stop1': { label: 'Stop' } } as any,
                dualButtonStates: { '/ops/Start0': 'on' },
            },
            [
                { refId: 'A', command: '/ops/Start' },
                { refId: 'B', command: 'Stop' },
            ]
        );

        expect(result.commandForms['command-0']).toMatchObject({
            ...legacyForm,
            commandName: '/ops/Start',
        });
        expect(result.commandForms['command-1'].commandName).toBe('Stop');
        expect(result.dualButtonStates?.['command-0']).toBe('on');
        expect(result.commandForms['/ops/Start0']).toBeUndefined();
    });

    it.each([undefined, {}])('seeds missing command forms from targets when forms are %s', (commandForms) => {
        const result = migrate(
            { commandForms: commandForms as PanelOptions['commandForms'] },
            [{ refId: 'A', command: '/ops/Start' }]
        );

        expect(result.commandForms['command-0'].commandName).toBe('/ops/Start');
        expect(result.commandForms['command-0'].arguments).toEqual({});
    });

    it('migrates variable-panel settings without inventing a command', () => {
        const result = migrate(
            { commandForms: { undefined0: { variableToSet: 'target' } } as any },
            [],
            'jaops-varcommanding-panel'
        );
        expect(result.commandForms['command-0'].variableToSet).toBe('target');
        expect(result.commandForms['command-0'].commandName).toBe('');
    });

    it('preserves ambiguous legacy keys when command names end in digits', () => {
        const targets = Array.from({ length: 11 }, (_, index) => ({
            refId: String.fromCharCode(65 + index),
            command: `Other${index}`,
        }));
        targets[0].command = 'Start1';
        targets[10].command = 'Start';
        const legacyForm = { label: 'Keep these arguments', arguments: { mode: 3 } };
        const result = migrate({ commandForms: { Start10: legacyForm } as any }, targets);

        expect(result.commandForms.Start10).toEqual(legacyForm);
        expect(result.commandForms['command-0'].commandName).toBe('Start1');
        expect(result.commandForms['command-10'].commandName).toBe('Start');
        expect(result.commandForms['command-0'].arguments).toEqual({});
        expect(result.commandForms['command-10'].arguments).toEqual({});
    });

    it('keeps legacy settings when the current form already exists', () => {
        const currentForm = {
            ...migrate({ commandForms: {} }, [{ refId: 'A', command: '/ops/Current' }]).commandForms['command-0'],
            label: 'Current',
        };
        const legacyForm = { label: 'Legacy', arguments: { mode: 2 } };
        const result = migrate(
            { commandForms: { 'command-0': currentForm, '/ops/Old0': legacyForm } as any },
            [{ refId: 'A', command: '/ops/Old' }]
        );

        expect(result.commandForms['command-0']).toBe(currentForm);
        expect(result.commandForms['/ops/Old0']).toEqual(legacyForm);
    });
});
