import { DataFrame } from '@grafana/data';

export function readLatestYamcsTime(series: DataFrame[]): number | null {
        if (!series || series.length === 0) {
            return null;
        }

        let newest: number | null = null;

        for (const frame of series) {
            const currentTimeField = frame.fields.find((field) => field.name === 'current_time');
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

            if (newest == null || timestamp > newest) {
                newest = timestamp;
            }
        }

        return newest;
}
