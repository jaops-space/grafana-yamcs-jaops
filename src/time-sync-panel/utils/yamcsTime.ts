import { DataFrame } from '@grafana/data';

export function readLatestYamcsTime(series: DataFrame[]): number | null {
    if (!series || series.length === 0) {
        return null;
    }

    let fallbackNewest: number | null = null;

    for (const [frameIndex, frame] of series.entries()) {
        const currentTimeField = frame.fields.find((field) => field.name === 'time');
        if (!currentTimeField || currentTimeField.values.length === 0) {
            continue;
        }

        const values = currentTimeField.values as unknown as ArrayLike<unknown>;
        const lastValue = values[values.length - 1];
        if (lastValue == null) {
            continue;
        }

        const dateValue = new Date(lastValue as any);
        const timestamp = dateValue.getTime();
        if (Number.isNaN(timestamp)) {
            continue;
        }

        // Prefer the first valid current_time frame (typically query A) so replay
        // time is not masked by newer timestamps from unrelated extra series.
        if (frameIndex === 0) {
            return timestamp;
        }

        if (fallbackNewest == null || timestamp > fallbackNewest) {
            fallbackNewest = timestamp;
        }
    }

    return fallbackNewest;
}
