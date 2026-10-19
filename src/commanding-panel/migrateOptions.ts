import { PanelMigrationHandler } from '@grafana/data';
import { PanelOptions } from './types';

export const migrateCommandOptions: PanelMigrationHandler<PanelOptions> = ({ options, targets }) => {
    const forms = options.commandForms;
    if (!forms) {
        return options;
    }

    const commandForms = { ...forms };
    const dualButtonStates = { ...options.dualButtonStates };
    for (const [key, form] of Object.entries(forms)) {
        if (/^command-\d+$/.test(key)) {
            continue;
        }
        const match = key.match(/^(.*?)(\d+)$/);
        if (!match) {
            continue;
        }
        const migratedKey = `command-${match[2]}`;
        if (!commandForms[migratedKey]) {
            const target = targets?.[Number(match[2])];
            const targetCommand = target && 'command' in target ? target.command : undefined;
            const legacyName = match[1] === 'undefined' ? '' : match[1];
            commandForms[migratedKey] = {
                ...form,
                commandName: form.commandName ?? (typeof targetCommand === 'string' ? targetCommand : legacyName),
            };
        }
        if (dualButtonStates[key] && !dualButtonStates[migratedKey]) {
            dualButtonStates[migratedKey] = dualButtonStates[key];
        }
        delete commandForms[key];
        delete dualButtonStates[key];
    }

    return { ...options, commandForms, dualButtonStates };
};
