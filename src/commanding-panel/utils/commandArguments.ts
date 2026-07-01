import { getTemplateSrv } from '@grafana/runtime';

export function coerceInputValue(value: string, engType: string): any {
    if (value.includes('$') || value.includes('{')) {
        return value;
    }
    if (engType === 'integer') {
        const parsed = parseInt(value, 10);
        return Number.isNaN(parsed) ? value : parsed;
    }
    if (engType === 'float') {
        const parsed = parseFloat(value);
        return Number.isNaN(parsed) ? value : parsed;
    }
    return value;
}

export function resolveArguments(args: Record<string, any> | undefined, commandInfo: any, scopedVars: any) {
    const resolvedArguments: Record<string, any> = {};
    if (!args) {
        return resolvedArguments;
    }

    const argTypeLookup: Record<string, string> = {};
    (commandInfo?.argument ?? []).forEach((arg: any) => {
        argTypeLookup[arg.name] = arg.type?.engType ?? 'string';
    });

    Object.entries(args).forEach(([argName, argValue]) => {
        const engType = argTypeLookup[argName] ?? 'string';

        if (typeof argValue !== 'string') {
            resolvedArguments[argName] = argValue;
            return;
        }

        const resolvedValue = getTemplateSrv().replace(argValue, scopedVars);
        if (engType === 'integer') {
            const parsed = parseInt(resolvedValue, 10);
            resolvedArguments[argName] = Number.isNaN(parsed) ? resolvedValue : parsed;
        } else if (engType === 'float') {
            const parsed = parseFloat(resolvedValue);
            resolvedArguments[argName] = Number.isNaN(parsed) ? resolvedValue : parsed;
        } else if (engType === 'boolean') {
            resolvedArguments[argName] =
                resolvedValue === 'true' ? true : resolvedValue === 'false' ? false : resolvedValue;
        } else {
            resolvedArguments[argName] = resolvedValue;
        }
    });

    return resolvedArguments;
}
