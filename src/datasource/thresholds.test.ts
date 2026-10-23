import {
    DataFrame,
    DataQueryResponse,
    FieldType,
    ThresholdsMode,
    applyFieldOverrides,
    createTheme,
} from '@grafana/data';
import { normalizeAlarmThresholds } from './thresholds';

function responseWithSteps(steps: Array<{ value: number | null; color: string }>): DataQueryResponse {
    // Exercise the real JSON wire format, where the base value is null.
    const frame: DataFrame = JSON.parse(JSON.stringify({
        name: '/Rover/temperature_front',
        length: 3,
        fields: [{
            name: '/Rover/temperature_front',
            type: FieldType.number,
            values: [-40, 0, 90],
            config: {
                color: { mode: 'thresholds' },
                thresholds: { mode: ThresholdsMode.Absolute, steps },
            },
        }],
    }));
    return { data: [frame] };
}

describe('Yamcs alarm thresholds over Grafana Live', () => {
    it.each([
        { mode: 'dark' as const, color: '#ccccdc' },
        { mode: 'light' as const, color: '#24292e' },
    ])('keeps unmonitored values neutral in the $mode theme', ({ mode, color }) => {
        const source = responseWithSteps([{ value: null, color: 'text' }]);
        const result = normalizeAlarmThresholds(source);
        expect(source.data[0].fields[0].config.thresholds.steps[0].value).toBeNull();
        expect(result.data[0].fields[0].config.thresholds.steps).toEqual([
            { value: -Infinity, color: 'text' },
        ]);
        const theme = createTheme({ colors: { mode } });
        const frames = applyFieldOverrides({
            data: result.data,
            fieldConfig: { defaults: {}, overrides: [] },
            theme,
            replaceVariables: (value) => value,
        });
        const field = frames[0].fields[0];
        expect(field.config.thresholds?.steps).toHaveLength(1);
        for (const value of [-40, 0, 90]) {
            expect(field.display?.(value).color).toBe(color);
        }
    });

    it('preserves every MDB boundary and severity color', () => {
        const source = responseWithSteps([
            { value: null, color: 'red' },
            { value: -20, color: 'yellow' },
            { value: 0, color: 'green' },
            { value: 50, color: 'yellow' },
            { value: 70, color: 'red' },
        ]);
        const result = normalizeAlarmThresholds(source);
        expect(result.data[0].fields[0].config.thresholds.steps).toEqual([
            { value: -Infinity, color: 'red' },
            ...source.data[0].fields[0].config.thresholds.steps.slice(1),
        ]);
        expect(result.data[0].fields[0].values).toBe(source.data[0].fields[0].values);
    });

    it('leaves normalized frames, non-parameter fields and empty responses unchanged', () => {
        const normalized = normalizeAlarmThresholds(responseWithSteps([{ value: null, color: 'text' }]));
        expect(normalizeAlarmThresholds(normalized).data[0]).toBe(normalized.data[0]);
        const other: DataFrame = {
            length: 0,
            fields: [{ name: 'time', type: FieldType.time, values: [], config: {} }],
        };
        expect(normalizeAlarmThresholds({ data: [other] }).data[0]).toBe(other);
        expect(normalizeAlarmThresholds({ data: [] }).data).toEqual([]);
    });
});
