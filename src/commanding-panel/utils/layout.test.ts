import { getRuntimeButtonWrapperStyle, getRuntimeLayoutError, getRuntimeLayoutStyle } from './layout';

describe('directional pad layout', () => {
    const options = { layoutDirection: 'direction-pad' as const, buttonMinWidth: 120, buttonHeight: 48, layoutGap: 8 };

    it('places forward/left/stop/right/reverse in a cross, with stop centered', () => {
        const positions = [0, 1, 2, 3, 4].map((index) => {
            const style = getRuntimeButtonWrapperStyle(options, index);
            return [style.gridRow, style.gridColumn];
        });
        expect(positions).toEqual([
            [1, 2],
            [2, 1],
            [2, 2],
            [2, 3],
            [3, 2],
        ]);
        expect(getRuntimeLayoutStyle(options)).toMatchObject({
            display: 'grid',
            gridTemplateColumns: 'repeat(3, minmax(120px, 1fr))',
            gridTemplateRows: 'repeat(3, 48px)',
            gap: '8px',
            overflow: 'auto',
        });
    });

    it('reports incomplete or extra command configurations instead of rendering misleading controls', () => {
        expect(getRuntimeLayoutError(options, 5)).toBeUndefined();
        for (const count of [0, 1, 4, 6]) {
            expect(getRuntimeLayoutError(options, count)).toMatch(/five commands/);
        }
    });

    it('preserves existing horizontal and vertical layouts', () => {
        expect(getRuntimeLayoutStyle({})).toMatchObject({ display: 'flex', flexDirection: 'column' });
        expect(getRuntimeLayoutStyle({ layoutDirection: 'row' })).toMatchObject({
            display: 'flex',
            flexDirection: 'row',
        });
        expect(getRuntimeButtonWrapperStyle({})).toMatchObject({ width: '100%', minWidth: 160, minHeight: 40 });
        expect(getRuntimeLayoutError({ layoutDirection: 'row' }, 6)).toBeUndefined();
    });
});
