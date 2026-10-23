import { DataFrame, DataQueryResponse } from '@grafana/data';

export function normalizeAlarmThresholds(response: DataQueryResponse): DataQueryResponse {
    return {
        ...response,
        data: response.data.map((frame: DataFrame) => {
            let changed = false;
            const fields = frame.fields.map((field) => {
                const thresholds = field.config.thresholds;
                if (thresholds?.steps[0]?.value !== null) {
                    return field;
                }
                // The Go SDK serializes -Infinity as null; Grafana Live leaves it
                // untouched, causing field overrides to insert a spurious green base.
                changed = true;
                return {
                    ...field,
                    config: {
                        ...field.config,
                        thresholds: {
                            ...thresholds,
                            steps: [
                                { ...thresholds.steps[0], value: Number.NEGATIVE_INFINITY },
                                ...thresholds.steps.slice(1),
                            ],
                        },
                    },
                };
            });
            return changed ? { ...frame, fields } : frame;
        }),
    };
}
