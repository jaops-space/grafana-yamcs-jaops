import { expectedParameters, minimumBackendStreams, panelQueries } from './workload';

const parameters = Array.from({ length: 131 }, (_, index) => `/test/value${index}`);
const indices = Array.from({ length: 100 }, (_, index) => index);

describe('canonical browser workload', () => {
    it('retains single queries alongside multi-parameter and multi-query panels', () => {
        const panels = indices.map((index) => panelQueries(index, parameters));
        expect(panels.some((queries) => queries.length === 1 && queries[0].length === 1)).toBe(true);
        expect(panels.some((queries) => queries.length > 1 && queries.some((query) => query.length > 1))).toBe(true);
        expect(expectedParameters(indices, parameters).sort()).toEqual([...parameters].sort());
    });

    it('shares identical queries between panels without losing parameter diversity', () => {
        expect(panelQueries(0, parameters)).toEqual(panelQueries(1, parameters));
        expect(panelQueries(10, parameters)).toEqual(panelQueries(11, parameters));
        expect(panelQueries(0, parameters)).toEqual(panelQueries(96, parameters));
        expect(minimumBackendStreams(indices, parameters)).toBeGreaterThan(1);
        expect(minimumBackendStreams(indices, parameters)).toBeLessThan(100);
    });

    it('does not duplicate members when using a reduced local pool', () => {
        for (const index of indices) {
            for (const query of panelQueries(index, parameters.slice(0, 3))) {
                expect(query.length).toBe(new Set(query).size);
            }
        }
    });
});
