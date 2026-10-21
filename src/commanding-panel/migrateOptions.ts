import { PanelMigrationHandler } from '@grafana/data';
import { PanelOptions } from './types';

function defaultCommandForm(commandName: string): CommandForm {
    return {
        commandName,
        arguments: {},
        comment: '',
        variableMode: false,
        variableToSet: '',
        changeMode: 'change',
        valueToSet: '',
        label: '',
        tooltip: '',
        icon: '',
        size: 'md',
        color: '',
        textColor: '',
        transparent: 'solid',
        shape: 'rectangle',
        customSVG: '',
        bgSize: '',
        bgPosition: '',
        bgWidth: '',
        bgHeight: '',
    };
}

type CommandForm = PanelOptions['commandForms'][string];
type MigrationTarget = { index: number; commandName: string };

export const migrateCommandOptions: PanelMigrationHandler<PanelOptions> = ({ options, targets, type }) => {
    const isVariablePanel = type === 'jaops-varcommanding-panel';
    const commandForms = { ...(options.commandForms ?? {}) };
    const dualButtonStates = { ...options.dualButtonStates };
    const targetCount = isVariablePanel ? Math.max(targets?.length ?? 0, 1) : targets?.length ?? 0;
    const migrationTargets: MigrationTarget[] = [];
    const targetsByLegacyKey = new Map<string, MigrationTarget[]>();

    for (let index = 0; index < targetCount; index++) {
        const target = targets?.[index];
        const command = target && 'command' in target && typeof target.command === 'string' ? target.command : '';
        const legacyKey = command ? `${command}${index}` : isVariablePanel ? `undefined${index}` : '';
        if (!legacyKey) {
            continue;
        }

        const migrationTarget = { index, commandName: command };
        migrationTargets.push(migrationTarget);
        const matches = targetsByLegacyKey.get(legacyKey) ?? [];
        matches.push(migrationTarget);
        targetsByLegacyKey.set(legacyKey, matches);
    }

    for (const [legacyKey, form] of Object.entries(options.commandForms ?? {})) {
        if (/^command-\d+$/.test(legacyKey)) {
            continue;
        }
        const matches = targetsByLegacyKey.get(legacyKey);
        if (matches?.length !== 1) {
            continue;
        }

        const { index, commandName } = matches[0];
        const currentKey = `command-${index}`;
        if (commandForms[currentKey]) {
            continue;
        }

        commandForms[currentKey] = {
            ...defaultCommandForm(commandName),
            ...form,
            commandName: form.commandName ?? commandName,
        };
        if (dualButtonStates[legacyKey] && !dualButtonStates[currentKey]) {
            dualButtonStates[currentKey] = dualButtonStates[legacyKey];
        }
        delete commandForms[legacyKey];
        delete dualButtonStates[legacyKey];
    }

    for (const { index, commandName } of migrationTargets) {
        const currentKey = `command-${index}`;
        if (!commandForms[currentKey]) {
            commandForms[currentKey] = defaultCommandForm(commandName);
        }
    }

    return { ...options, commandForms, dualButtonStates };
};
